import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const TURNSTILE_SCRIPT_URL =
  'https://challenges.cloudflare.com/turnstile/v0/api.js';
const TURNSTILE_SCRIPT_SELECTOR = 'script[data-newapi-turnstile="true"]';
const SCRIPT_LOAD_TIMEOUT = 10000;
const SLOW_VERIFICATION_TIMEOUT = 12000;
const SCRIPT_LOAD_ATTEMPTS = 2;

let turnstileLoadPromise = null;
let turnstileLoadGeneration = 0;
let turnstileCallbackSequence = 0;
const pendingScriptLoads = new Set();

const createCancelledError = () => {
  const error = new Error('Turnstile script load cancelled');
  error.name = 'AbortError';
  return error;
};

const delay = (milliseconds) =>
  new Promise((resolve) => setTimeout(resolve, milliseconds));

const loadTurnstileScriptOnce = () =>
  new Promise((resolve, reject) => {
    if (globalThis.turnstile?.render) {
      resolve(globalThis.turnstile);
      return;
    }

    document
      .querySelectorAll(TURNSTILE_SCRIPT_SELECTOR)
      .forEach((script) => script.remove());

    const callbackName = `__newApiTurnstileOnLoad_${Date.now()}_${turnstileCallbackSequence++}`;
    const script = document.createElement('script');
    let settled = false;
    let timeoutId;
    let cancel;

    const cleanup = () => {
      clearTimeout(timeoutId);
      delete globalThis[callbackName];
      pendingScriptLoads.delete(cancel);
    };
    const fail = (error) => {
      if (settled) return;
      settled = true;
      cleanup();
      script.remove();
      reject(error);
    };
    timeoutId = setTimeout(() => {
      fail(new Error('Turnstile script load timed out'));
    }, SCRIPT_LOAD_TIMEOUT);
    cancel = () => fail(createCancelledError());

    pendingScriptLoads.add(cancel);

    globalThis[callbackName] = () => {
      if (settled) return;
      if (!globalThis.turnstile?.render) {
        fail(new Error('Turnstile API is unavailable after script load'));
        return;
      }
      settled = true;
      cleanup();
      resolve(globalThis.turnstile);
    };

    script.dataset.newapiTurnstile = 'true';
    script.async = true;
    script.defer = true;
    script.src = `${TURNSTILE_SCRIPT_URL}?onload=${callbackName}&render=explicit`;
    script.addEventListener('error', () => {
      fail(new Error('Failed to load Turnstile script'));
    });
    document.head.appendChild(script);
  });

const loadTurnstile = () => {
  if (globalThis.turnstile?.render) {
    return Promise.resolve(globalThis.turnstile);
  }
  if (turnstileLoadPromise) {
    return turnstileLoadPromise;
  }

  const loadGeneration = turnstileLoadGeneration;
  const loadPromise = (async () => {
    let lastError;
    for (let attempt = 0; attempt < SCRIPT_LOAD_ATTEMPTS; attempt += 1) {
      if (loadGeneration !== turnstileLoadGeneration) {
        throw createCancelledError();
      }
      try {
        return await loadTurnstileScriptOnce();
      } catch (error) {
        if (error.name === 'AbortError') throw error;
        lastError = error;
        if (attempt + 1 < SCRIPT_LOAD_ATTEMPTS) {
          await delay(1000 * (attempt + 1));
        }
      }
    }
    throw lastError;
  })();
  turnstileLoadPromise = loadPromise;
  void loadPromise.catch(() => {
    if (turnstileLoadPromise === loadPromise) {
      turnstileLoadPromise = null;
    }
  });

  return turnstileLoadPromise;
};

const restartTurnstileLoader = () => {
  if (globalThis.turnstile?.render) return;
  turnstileLoadGeneration += 1;
  turnstileLoadPromise = null;
  [...pendingScriptLoads].forEach((cancel) => cancel());
  document
    .querySelectorAll(TURNSTILE_SCRIPT_SELECTOR)
    .forEach((script) => script.remove());
};

const ResilientTurnstile = ({
  sitekey,
  action,
  cData,
  theme,
  language,
  tabIndex,
  responseField,
  responseFieldName,
  size,
  fixedSize = false,
  retry,
  retryInterval,
  refreshExpired,
  appearance,
  execution,
  id,
  userRef,
  className = '',
  style,
  onVerify,
  onSuccess,
  onLoad,
  onError,
  onExpire,
  onTimeout,
  onAfterInteractive,
  onBeforeInteractive,
  onUnsupported,
}) => {
  const { t } = useTranslation();
  const ownContainerRef = useRef(null);
  const containerRef = userRef ?? ownContainerRef;
  const widgetRef = useRef(null);
  const callbacksRef = useRef({});
  const [renderAttempt, setRenderAttempt] = useState(0);
  const [status, setStatus] = useState('loading');
  const [isSlow, setIsSlow] = useState(false);

  callbacksRef.current = {
    onVerify,
    onSuccess,
    onLoad,
    onError,
    onExpire,
    onTimeout,
    onAfterInteractive,
    onBeforeInteractive,
    onUnsupported,
  };

  useEffect(() => {
    if (status !== 'loading' && status !== 'checking') {
      setIsSlow(false);
      return undefined;
    }
    const timer = setTimeout(() => setIsSlow(true), SLOW_VERIFICATION_TIMEOUT);
    return () => clearTimeout(timer);
  }, [status]);

  useEffect(() => {
    let cancelled = false;
    let widgetId = null;
    let turnstileApi;
    const container = containerRef.current;

    setStatus('loading');
    setIsSlow(false);
    widgetRef.current = null;

    const renderWidget = async () => {
      if (!sitekey) {
        throw new Error('Turnstile site key is empty');
      }

      turnstileApi = await loadTurnstile();
      if (cancelled || !container) return;

      const boundTurnstile = {
        execute: (executeOptions) => {
          setStatus('checking');
          return turnstileApi.execute(widgetId, executeOptions);
        },
        reset: () => {
          setStatus('checking');
          setIsSlow(false);
          return turnstileApi.reset(widgetId);
        },
        getResponse: () => turnstileApi.getResponse(widgetId),
        isExpired: () => turnstileApi.isExpired(widgetId),
      };
      const options = {
        sitekey,
        action,
        cData,
        theme,
        language,
        tabindex: tabIndex,
        'response-field': responseField,
        'response-field-name': responseFieldName,
        size,
        retry,
        'retry-interval': retryInterval,
        'refresh-expired': refreshExpired,
        appearance,
        execution,
        callback: (token, preClearanceObtained) => {
          if (cancelled) return;
          setStatus('verified');
          callbacksRef.current.onVerify?.(token, boundTurnstile);
          callbacksRef.current.onSuccess?.(
            token,
            preClearanceObtained,
            boundTurnstile,
          );
        },
        'error-callback': (error) => {
          if (cancelled) return;
          setStatus('error');
          callbacksRef.current.onError?.(error, boundTurnstile);
        },
        'expired-callback': (token) => {
          if (cancelled) return;
          setStatus('checking');
          callbacksRef.current.onExpire?.(token, boundTurnstile);
        },
        'timeout-callback': () => {
          if (cancelled) return;
          setStatus('error');
          callbacksRef.current.onTimeout?.(boundTurnstile);
        },
        'after-interactive-callback': () => {
          if (!cancelled) {
            callbacksRef.current.onAfterInteractive?.(boundTurnstile);
          }
        },
        'before-interactive-callback': () => {
          if (!cancelled) {
            callbacksRef.current.onBeforeInteractive?.(boundTurnstile);
          }
        },
        'unsupported-callback': () => {
          if (cancelled) return;
          setStatus('error');
          callbacksRef.current.onUnsupported?.(boundTurnstile);
        },
      };

      Object.keys(options).forEach((key) => {
        if (options[key] === undefined) delete options[key];
      });

      container.replaceChildren();
      widgetId = turnstileApi.render(container, options);
      if (widgetId === undefined || widgetId === null) {
        throw new Error('Turnstile widget failed to render');
      }
      widgetRef.current = { api: turnstileApi, id: widgetId };
      callbacksRef.current.onLoad?.(widgetId, boundTurnstile);
    };

    renderWidget().catch((error) => {
      if (cancelled) return;
      if (error.name === 'AbortError') {
        setRenderAttempt((attempt) => attempt + 1);
        return;
      }
      setStatus('error');
      callbacksRef.current.onError?.(error);
    });

    return () => {
      cancelled = true;
      widgetRef.current = null;
      if (widgetId !== null && turnstileApi?.remove) {
        try {
          turnstileApi.remove(widgetId);
        } catch (error) {
          void error;
        }
      }
      container?.replaceChildren();
    };
  }, [
    sitekey,
    action,
    cData,
    theme,
    language,
    tabIndex,
    responseField,
    responseFieldName,
    size,
    retry,
    retryInterval,
    refreshExpired,
    appearance,
    execution,
    userRef,
    renderAttempt,
  ]);

  const retryVerification = useCallback(() => {
    setIsSlow(false);
    const widget = widgetRef.current;
    if (widget?.api?.reset) {
      setStatus('checking');
      try {
        widget.api.reset(widget.id);
        return;
      } catch {
        widgetRef.current = null;
      }
    }
    restartTurnstileLoader();
    setRenderAttempt((attempt) => attempt + 1);
  }, []);

  const fixedDimensions = fixedSize
    ? {
        width: size === 'compact' ? 130 : size === 'flexible' ? '100%' : 300,
        height: size === 'compact' ? 120 : 65,
      }
    : {};
  const showProgress = status === 'loading' || status === 'checking';
  const showRecovery = status === 'error' || isSlow;

  return (
    <div className='flex w-full flex-col items-center'>
      <div
        ref={containerRef}
        id={id}
        className={className}
        style={{ ...fixedDimensions, ...style }}
      />
      {showProgress && !showRecovery && (
        <div
          className='mt-1 flex items-center gap-2 text-xs text-gray-500'
          role='status'
          aria-live='polite'
        >
          <Spin size='small' />
          <span>{t('安全验证')}...</span>
        </div>
      )}
      {showRecovery && (
        <div
          className='mt-2 flex max-w-[320px] items-center gap-2 text-xs text-amber-700'
          role='alert'
        >
          <span>
            {status === 'error'
              ? `${t('安全验证')}：${t('加载失败')}`
              : `${t('安全验证')}：${t('加载中')}`}
          </span>
          <Button size='small' theme='borderless' onClick={retryVerification}>
            {t('重试')}
          </Button>
        </div>
      )}
    </div>
  );
};

export default ResilientTurnstile;
