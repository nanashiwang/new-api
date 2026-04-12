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

import React, { useState, useEffect, useRef } from 'react';
import {
  Notification,
  Button,
  Space,
  Toast,
  Select,
} from '@douyinfe/semi-ui';
import { API, showError, getModelCategories, selectFilter } from '../../helpers';

/**
 * 管理 FluentRead 浏览器扩展集成的所有状态和副作用，
 * 从 tokens/index.jsx 抽离，职责清晰。
 *
 * @param {object} params
 * @param {Array}  params.tokens       - 当前 token 列表（来自 useTokensData）
 * @param {Array}  params.selectedKeys - 已选中 token 列表
 * @param {Function} params.t          - 翻译函数
 * @returns {{ openFluentNotification: Function, modelOptions: Array }}
 */
export function useFluentIntegration({ tokens, selectedKeys, t }) {
  const [modelOptions, setModelOptions] = useState([]);
  const [selectedModel, setSelectedModel] = useState('');
  const [fluentNoticeOpen, setFluentNoticeOpen] = useState(false);
  const [prefillKey, setPrefillKey] = useState('');

  // 保存最新值，供 Notification 内的回调（onClick 等）读取，避免 stale closure
  const latestRef = useRef({
    tokens: [],
    selectedKeys: [],
    t: (k) => k,
    selectedModel: '',
    prefillKey: '',
  });

  // 保存最新版本的 openFluentNotification，供事件监听器调用
  const openFluentNotificationRef = useRef(null);

  useEffect(() => {
    latestRef.current = {
      tokens,
      selectedKeys,
      t,
      selectedModel,
      prefillKey,
    };
  }, [tokens, selectedKeys, t, selectedModel, prefillKey]);

  // 加载可用模型列表
  const loadModels = async () => {
    try {
      const res = await API.get('/api/user/models');
      const { success, message, data } = res.data || {};
      if (success) {
        const { t: currentT } = latestRef.current;
        const categories = getModelCategories(currentT);
        const options = (data || []).map((model) => {
          let icon = null;
          for (const [key, category] of Object.entries(categories)) {
            if (key !== 'all' && category.filter({ model_name: model })) {
              icon = category.icon;
              break;
            }
          }
          return {
            label: (
              <span className='flex items-center gap-1'>
                {icon}
                {model}
              </span>
            ),
            value: model,
          };
        });
        setModelOptions(options);
      } else {
        showError(latestRef.current.t(message));
      }
    } catch (e) {
      showError(e.message || 'Failed to load models');
    }
  };

  // Notification 中"一键填充"按钮的处理函数
  // 通过 latestRef 读取最新值，避免 Notification 渲染后 closure 过期
  const handlePrefillToFluent = () => {
    const {
      tokens: latestTokens,
      selectedKeys: latestSelectedKeys,
      t: latestT,
      selectedModel: chosenModel,
      prefillKey: overrideKey,
    } = latestRef.current;

    const container = document.getElementById('fluent-new-api-container');
    if (!container) {
      Toast.error(latestT('未检测到 Fluent 容器'));
      return;
    }

    if (!chosenModel) {
      Toast.warning(latestT('请选择模型'));
      return;
    }

    let status = localStorage.getItem('status');
    let serverAddress = '';
    if (status) {
      try {
        status = JSON.parse(status);
        serverAddress = status.server_address || '';
      } catch (_) {}
    }
    if (!serverAddress) serverAddress = window.location.origin;

    let apiKeyToUse = '';
    if (overrideKey) {
      apiKeyToUse = 'sk-' + overrideKey;
    } else {
      const token =
        latestSelectedKeys && latestSelectedKeys.length === 1
          ? latestSelectedKeys[0]
          : latestTokens && latestTokens.length > 0
            ? latestTokens[0]
            : null;
      if (!token) {
        Toast.warning(latestT('没有可用令牌用于填充'));
        return;
      }
      apiKeyToUse = 'sk-' + token.key;
    }

    const payload = {
      id: 'new-api',
      baseUrl: serverAddress,
      apiKey: apiKeyToUse,
      model: chosenModel,
    };

    container.dispatchEvent(
      new CustomEvent('fluent:prefill', { detail: payload }),
    );
    Toast.success(latestT('已发送到 Fluent'));
    Notification.close('fluent-detected');
  };

  // 打开 FluentRead 检测通知
  // 普通函数（非 useCallback），每次渲染重新定义，
  // 保证 modelOptions 等状态是最新值
  function openFluentNotification(key) {
    const { t: currentT } = latestRef.current;
    const SUPPRESS_KEY = 'fluent_notify_suppressed';

    if (modelOptions.length === 0) {
      // 触发加载后的 effect 会刷新通知内容
      loadModels();
    }
    if (!key && localStorage.getItem(SUPPRESS_KEY) === '1') return;

    const container = document.getElementById('fluent-new-api-container');
    if (!container) {
      Toast.warning(currentT('未检测到 FluentRead（流畅阅读），请确认扩展已启用'));
      return;
    }

    setPrefillKey(key || '');
    setFluentNoticeOpen(true);
    Notification.info({
      id: 'fluent-detected',
      title: currentT('检测到 FluentRead（流畅阅读）'),
      content: (
        <div>
          <div style={{ marginBottom: 8 }}>
            {key
              ? currentT('请选择模型。')
              : currentT('选择模型后可一键填充当前选中令牌（或本页第一个令牌）。')}
          </div>
          <div style={{ marginBottom: 8 }}>
            <Select
              placeholder={currentT('请选择模型')}
              optionList={modelOptions}
              onChange={setSelectedModel}
              filter={selectFilter}
              style={{ width: 320 }}
              showClear
              searchable
              emptyContent={currentT('暂无数据')}
            />
          </div>
          <Space>
            <Button
              theme='solid'
              type='primary'
              onClick={handlePrefillToFluent}
            >
              {currentT('一键填充到 FluentRead')}
            </Button>
            {!key && (
              <Button
                type='warning'
                onClick={() => {
                  localStorage.setItem(SUPPRESS_KEY, '1');
                  Notification.close('fluent-detected');
                  Toast.info(currentT('已关闭后续提醒'));
                }}
              >
                {currentT('不再提醒')}
              </Button>
            )}
            <Button
              type='tertiary'
              onClick={() => Notification.close('fluent-detected')}
            >
              {currentT('关闭')}
            </Button>
          </Space>
        </div>
      ),
      duration: 0,
    });
  }

  // 每次渲染后更新 ref，让事件监听器始终调用最新版本
  openFluentNotificationRef.current = openFluentNotification;

  // Fluent 容器出现/消失时的事件监听
  useEffect(() => {
    const onAppeared = () => {
      openFluentNotificationRef.current?.();
    };
    const onRemoved = () => {
      setFluentNoticeOpen(false);
      Notification.close('fluent-detected');
    };

    window.addEventListener('fluent-container:appeared', onAppeared);
    window.addEventListener('fluent-container:removed', onRemoved);
    return () => {
      window.removeEventListener('fluent-container:appeared', onAppeared);
      window.removeEventListener('fluent-container:removed', onRemoved);
    };
  }, []);

  // 模型列表或语言变化时，若通知已打开则刷新内容
  useEffect(() => {
    if (fluentNoticeOpen) {
      openFluentNotificationRef.current?.();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [modelOptions, selectedModel, t, fluentNoticeOpen]);

  // MutationObserver：监听 #fluent-new-api-container 的 DOM 挂载/卸载
  useEffect(() => {
    const selector = '#fluent-new-api-container';
    const root = document.body || document.documentElement;

    const existing = document.querySelector(selector);
    if (existing) {
      console.log('Fluent container detected (initial):', existing);
      window.dispatchEvent(
        new CustomEvent('fluent-container:appeared', { detail: existing }),
      );
    }

    const isOrContainsTarget = (node) => {
      if (!(node && node.nodeType === 1)) return false;
      if (node.id === 'fluent-new-api-container') return true;
      return (
        typeof node.querySelector === 'function' &&
        !!node.querySelector(selector)
      );
    };

    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        for (const added of m.addedNodes) {
          if (isOrContainsTarget(added)) {
            const el = document.querySelector(selector);
            if (el) {
              console.log('Fluent container appeared:', el);
              window.dispatchEvent(
                new CustomEvent('fluent-container:appeared', { detail: el }),
              );
            }
            break;
          }
        }
        for (const removed of m.removedNodes) {
          if (isOrContainsTarget(removed)) {
            const elNow = document.querySelector(selector);
            if (!elNow) {
              console.log('Fluent container removed');
              window.dispatchEvent(new CustomEvent('fluent-container:removed'));
            }
            break;
          }
        }
      }
    });

    observer.observe(root, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, []);

  return {
    openFluentNotification,
    modelOptions,
  };
}
