/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { Button, Tag, Typography } from '@douyinfe/semi-ui';
import { AlertTriangle, Clock3, ShieldAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API } from '../../helpers/apiCore';
import { UserContext } from '../../context/User';
import { timestamp2string } from '../../helpers';
import { formatContentSafetyCategory } from '../../helpers/contentSafety';

const { Text } = Typography;

const formatRemaining = (seconds) => {
  const safe = Math.max(0, Number(seconds || 0));
  const minutes = Math.floor(safe / 60);
  const remainder = safe % 60;
  return `${minutes}:${String(remainder).padStart(2, '0')}`;
};

const ContentSafetyNotice = () => {
  const { t } = useTranslation();
  const [userState] = useContext(UserContext);
  const [state, setState] = useState(null);
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  const [acknowledging, setAcknowledging] = useState(false);

  const loadState = useCallback(async () => {
    if (!userState?.user?.id) {
      setState(null);
      return;
    }
    try {
      const response = await API.get('/api/content-safety/self', {
        disableDuplicate: true,
        skipGlobalLoading: true,
        skipErrorHandler: true,
      });
      if (response?.data?.success) {
        setState(response.data.data || null);
      }
    } catch {
      // A warning lookup must never make the rest of the console unusable.
    }
  }, [userState?.user?.id]);

  useEffect(() => {
    loadState();
    const poll = window.setInterval(loadState, 30000);
    return () => window.clearInterval(poll);
  }, [loadState]);

  useEffect(() => {
    if (!state?.cooldown_until) return undefined;
    const timer = window.setInterval(
      () => setNow(Math.floor(Date.now() / 1000)),
      1000,
    );
    return () => window.clearInterval(timer);
  }, [state?.cooldown_until]);

  const cooldownRemaining = Math.max(
    0,
    Number(state?.cooldown_until || 0) - now,
  );
  const coolingOff = cooldownRemaining > 0;
  const visible = Boolean(
    coolingOff ||
      state?.has_unread_warning ||
      state?.level === 'review_pending',
  );
  const latest = state?.latest_violation;

  const tone = useMemo(() => {
    if (state?.level === 'review_pending') {
      return {
        border: '#b91c1c',
        background: '#fff1f2',
        icon: ShieldAlert,
        label: t('已提交管理员复核'),
      };
    }
    if (coolingOff) {
      return {
        border: '#dc2626',
        background: '#fff7ed',
        icon: Clock3,
        label: t('10 分钟冷静期'),
      };
    }
    return {
      border: '#d97706',
      background: '#fffbeb',
      icon: AlertTriangle,
      label: t('重要内容安全警告'),
    };
  }, [coolingOff, state?.level, t]);

  const acknowledge = async () => {
    setAcknowledging(true);
    try {
      const response = await API.post(
        '/api/content-safety/self/acknowledge',
        {},
        { skipGlobalLoading: true },
      );
      if (response?.data?.success) {
        setState((previous) =>
          previous ? { ...previous, has_unread_warning: false } : previous,
        );
      }
    } finally {
      setAcknowledging(false);
    }
  };

  if (!visible) return null;
  const Icon = tone.icon;

  return (
    <section
      role='alert'
      className='mx-2 mt-2 rounded-xl border-l-4 px-4 py-3 shadow-sm md:mx-6'
      style={{ borderColor: tone.border, background: tone.background }}
    >
      <div className='flex flex-col gap-3 md:flex-row md:items-start md:justify-between'>
        <div className='flex min-w-0 gap-3'>
          <Icon size={22} color={tone.border} className='mt-0.5 shrink-0' />
          <div className='min-w-0'>
            <div className='flex flex-wrap items-center gap-2'>
              <Text strong style={{ color: tone.border }}>
                {tone.label}
              </Text>
              {coolingOff ? (
                <Tag color='red' shape='circle'>
                  {t('剩余')} {formatRemaining(cooldownRemaining)}
                </Tag>
              ) : null}
              <Tag color='orange' shape='circle'>
                {t('近30天')} {Number(state?.window_count || 0)} {t('次')}
              </Tag>
            </div>
            <p className='mb-0 mt-1 text-sm leading-6 text-[var(--semi-color-text-0)]'>
              {state?.level === 'review_pending'
                ? t(
                    '由于30天内多次进入冷静期，系统已提交管理员人工复核。只有管理员明确批准后才会永久停用账户。',
                  )
                : coolingOff
                  ? t(
                      '冷静期内的新模型请求会在本地被拒绝，不会发送到上游，也不会新增风控次数。冷静期结束后自动恢复。',
                    )
                  : t(
                      '上游明确拒绝了最近一次请求。请调整请求内容；10 分钟内第三次确认拒绝将进入 10 分钟冷静期。',
                    )}
            </p>
            <div className='mt-2 grid gap-x-6 gap-y-1 text-xs text-[var(--semi-color-text-2)] md:grid-cols-2'>
              <span>
                {t('官方错误码')}: {latest?.error_code || '-'}
              </span>
              <span>
                {t('本地细分类')}:{' '}
                {formatContentSafetyCategory(latest?.fine_category, t)}
              </span>
              <span>
                {t('分类来源')}:{' '}
                {latest?.reason_source === 'local_rule'
                  ? t('本地规则推断')
                  : latest?.reason_source || '-'}
              </span>
              <span>
                {t('最近触发时间')}:{' '}
                {latest?.created_at ? timestamp2string(latest.created_at) : '-'}
              </span>
            </div>
            {latest?.reason_summary ? (
              <p className='mb-0 mt-2 text-xs leading-5 text-[var(--semi-color-text-1)]'>
                {latest.reason_summary}
              </p>
            ) : null}
          </div>
        </div>
        {state?.has_unread_warning ? (
          <Button
            type='warning'
            theme='solid'
            loading={acknowledging}
            onClick={acknowledge}
            className='shrink-0'
          >
            {t('我已知晓')}
          </Button>
        ) : null}
      </div>
    </section>
  );
};

export default ContentSafetyNotice;
