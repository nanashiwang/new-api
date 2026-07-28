/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Form,
  Modal,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  buildCCSwitchDeepLink,
  CC_SWITCH_APPS,
  chooseDefaultModel,
  getCompatibleModels,
  getRouteOptions,
} from '../../../../utils/ccSwitch';

const CCSwitchModal = ({ visible, token, getTokenFullKey, onCancel, t }) => {
  const isMobile = useIsMobile();
  const requestIdRef = useRef(0);
  const [loading, setLoading] = useState(false);
  const [models, setModels] = useState([]);
  const [apiKey, setApiKey] = useState('');
  const [app, setApp] = useState('codex');
  const [model, setModel] = useState('');
  const [route, setRoute] = useState('');

  const routeOptions = useMemo(() => {
    let configuredAddress = '';
    try {
      configuredAddress =
        JSON.parse(localStorage.getItem('status'))?.server_address || '';
    } catch (_) {
      configuredAddress = '';
    }
    return getRouteOptions(configuredAddress, window.location.origin);
  }, [visible]);

  useEffect(() => {
    if (!visible || !token?.id) return;
    const requestId = ++requestIdRef.current;
    setLoading(true);
    setModels([]);
    setApiKey('');
    setApp('codex');
    const previousRoute = localStorage.getItem('ccswitch:last-route') || '';
    setRoute(
      routeOptions.includes(previousRoute)
        ? previousRoute
        : routeOptions[0] || '',
    );
    Promise.all([
      API.get('/api/token/models', {
        params: { token_id: token.id, detail: true },
      }),
      getTokenFullKey(token),
    ])
      .then(([response, key]) => {
        if (requestId !== requestIdRef.current) return;
        const { success, message, data } = response.data || {};
        if (!success) throw new Error(message || t('加载模型失败'));
        if (!key) throw new Error(t('获取令牌失败'));
        const nextModels = Array.isArray(data) ? data : [];
        setModels(nextModels);
        setApiKey(key.startsWith('sk-') ? key : `sk-${key}`);
        setModel(
          chooseDefaultModel(
            nextModels,
            'codex',
            localStorage.getItem('ccswitch:last-model:codex') || '',
          ),
        );
      })
      .catch((error) => {
        if (requestId === requestIdRef.current) {
          showError(error?.message || t('加载 CC Switch 配置失败'));
        }
      })
      .finally(() => {
        if (requestId === requestIdRef.current) setLoading(false);
      });
    return () => {
      requestIdRef.current += 1;
      setApiKey('');
    };
  }, [visible, token?.id]);

  useEffect(() => {
    setModel((current) =>
      chooseDefaultModel(
        models,
        app,
        localStorage.getItem(`ccswitch:last-model:${app}`) || current,
      ),
    );
  }, [app, models]);

  const compatibleModels = useMemo(
    () => getCompatibleModels(models, app),
    [models, app],
  );
  const isModelCompatible = compatibleModels.some(
    (item) => item.name === model,
  );

  const launch = () => {
    if (!apiKey || !route || !model || !isModelCompatible) {
      showError(t('当前令牌没有适用于该客户端的模型，无法导入'));
      return;
    }
    try {
      const link = buildCCSwitchDeepLink({
        app,
        name: `${token?.name || 'new-api'} · ${CC_SWITCH_APPS[app].label}`,
        baseUrl: route,
        apiKey,
        model,
      });
      window.location.href = link;
      localStorage.setItem(`ccswitch:last-model:${app}`, model);
      localStorage.setItem('ccswitch:last-route', route);
      showSuccess(t('已请求打开 CC Switch，请在客户端确认导入'));
    } catch (error) {
      showError(error?.message || t('生成 CC Switch 配置失败'));
    }
  };

  return (
    <Modal
      title={t('一键配置 CC Switch')}
      visible={visible}
      onCancel={onCancel}
      width={isMobile ? 'calc(100vw - 24px)' : 560}
      footer={
        <>
          <Button onClick={onCancel}>{t('取消')}</Button>
          <Button
            theme='solid'
            type='primary'
            disabled={
              loading || !apiKey || !route || !model || !isModelCompatible
            }
            onClick={launch}
          >
            {t('打开 CC Switch 并自动导入')}
          </Button>
        </>
      }
    >
      <Spin spinning={loading}>
        <Banner
          type='info'
          description={t(
            '配置只会发送给你本机的 CC Switch；页面不会展示密钥。重复导入可能生成同名 Provider。',
          )}
          className='mb-4'
        />
        <Form layout='vertical'>
          <Form.Select
            field='app'
            label={t('目标应用')}
            value={app}
            onChange={setApp}
            optionList={Object.entries(CC_SWITCH_APPS).map(([value, item]) => ({
              value,
              label: item.label,
            }))}
            style={{ width: '100%' }}
          />
          <Form.Select
            field='model'
            label={t('默认模型')}
            value={model || undefined}
            onChange={setModel}
            optionList={compatibleModels.map((item) => ({
              value: item.name,
              label: item.name,
            }))}
            emptyContent={t('当前令牌没有兼容模型')}
            filter
            style={{ width: '100%' }}
          />
          <Form.Select
            field='route'
            label={t('API 线路')}
            value={route || undefined}
            onChange={setRoute}
            optionList={routeOptions.map((value, index) => ({
              value,
              label: index === 0 ? `${value} (${t('推荐')})` : value,
            }))}
            style={{ width: '100%' }}
          />
        </Form>
        {!loading && compatibleModels.length === 0 && (
          <Banner
            type='warning'
            description={t(
              '该令牌当前没有适用于所选客户端的文本模型。请更换令牌分组或选择其他客户端。',
            )}
          />
        )}
        <Typography.Text type='tertiary' size='small'>
          {t('导入后仍会由 CC Switch 显示最终配置并要求你确认。')}
        </Typography.Text>
      </Spin>
    </Modal>
  );
};

export default CCSwitchModal;
