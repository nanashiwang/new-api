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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Collapse,
  Form,
  Row,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, Save } from 'lucide-react';
import {
  API,
  compareObjects,
  showError,
  showSuccess,
  showWarning,
  toBoolean,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const DEFAULT_INPUTS = {
  'channel_concurrency_setting.enabled': false,
  'channel_concurrency_setting.default_max_concurrency': 100,
  'channel_concurrency_setting.default_rpm_limit': 0,
  'channel_concurrency_setting.rpm_window_seconds': 60,
  'channel_concurrency_setting.wait_timeout_ms': 3000,
  'channel_concurrency_setting.max_queue_length': 200,
  'channel_concurrency_setting.poll_interval_ms': 50,
  'channel_concurrency_setting.retry_after_seconds': 3,
  'channel_concurrency_setting.affinity_sticky_enabled': true,
  'channel_concurrency_setting.affinity_wait_ms': 2000,
};

const BOOLEAN_KEYS = new Set([
  'channel_concurrency_setting.enabled',
  'channel_concurrency_setting.affinity_sticky_enabled',
]);

const NUMERIC_RULES = {
  'channel_concurrency_setting.default_max_concurrency': {
    min: 1,
    label: '全局默认单渠道并发',
  },
  'channel_concurrency_setting.default_rpm_limit': {
    min: 0,
    label: '全局默认单渠道 RPM',
  },
  'channel_concurrency_setting.rpm_window_seconds': {
    min: 1,
    label: 'RPM 统计窗口',
  },
  'channel_concurrency_setting.wait_timeout_ms': {
    min: 1,
    label: '全部渠道满载等待时间',
  },
  'channel_concurrency_setting.max_queue_length': {
    min: 1,
    label: '等待队列上限',
  },
  'channel_concurrency_setting.poll_interval_ms': {
    min: 10,
    label: '容量探测间隔',
  },
  'channel_concurrency_setting.retry_after_seconds': {
    min: 1,
    label: '客户端重试提示',
  },
  'channel_concurrency_setting.affinity_wait_ms': {
    min: 1,
    label: '亲和渠道等待时间',
  },
};

const parseOptions = (options) => {
  const next = { ...DEFAULT_INPUTS };
  for (const key of Object.keys(DEFAULT_INPUTS)) {
    if (!(key in (options || {}))) continue;
    if (BOOLEAN_KEYS.has(key)) {
      next[key] = toBoolean(options[key]);
      continue;
    }
    const value = Number(options[key]);
    const min = NUMERIC_RULES[key]?.min ?? 0;
    next[key] =
      Number.isInteger(value) && value >= min ? value : DEFAULT_INPUTS[key];
  }
  return next;
};

export default function SettingsChannelConcurrency(props) {
  const { t } = useTranslation();
  const { Text } = Typography;
  const formRef = useRef();
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [inputs, setInputs] = useState(DEFAULT_INPUTS);
  const [savedInputs, setSavedInputs] = useState(DEFAULT_INPUTS);
  const [runtime, setRuntime] = useState(null);

  const setField = (key) => (value) => {
    setInputs((current) => ({ ...current, [key]: value }));
  };

  const refreshRuntime = async () => {
    try {
      setStatsLoading(true);
      const response = await API.get('/api/channel_concurrency/stats', {
        disableDuplicate: true,
      });
      const { success, message, data } = response.data;
      if (!success) return showError(t(message || '刷新运行状态失败'));
      setRuntime(data || null);
    } catch (error) {
      showError(t('刷新渠道并发运行状态失败'));
    } finally {
      setStatsLoading(false);
    }
  };

  const validateInputs = () => {
    for (const [key, rule] of Object.entries(NUMERIC_RULES)) {
      const value = Number(inputs[key]);
      if (!Number.isInteger(value) || value < rule.min) {
        showError(
          t('{{label}}必须是不小于 {{min}} 的整数', {
            label: t(rule.label),
            min: rule.min,
          }),
        );
        return false;
      }
    }

    if (
      inputs['channel_concurrency_setting.affinity_sticky_enabled'] &&
      Number(inputs['channel_concurrency_setting.affinity_wait_ms']) >
        Number(inputs['channel_concurrency_setting.wait_timeout_ms'])
    ) {
      showError(t('亲和渠道等待时间不能大于全部渠道满载等待时间'));
      return false;
    }
    return true;
  };

  const onSubmit = async () => {
    if (!validateInputs()) return;
    const changes = compareObjects(inputs, savedInputs);
    if (!changes.length) return showWarning(t('你似乎并没有修改什么'));

    try {
      setLoading(true);
      const responses = await Promise.all(
        changes.map(({ key }) =>
          API.put('/api/option/', {
            key,
            value: String(inputs[key]),
          }),
        ),
      );
      if (responses.some((response) => !response?.data?.success)) {
        const failed = responses.find(
          (response) => response?.data && !response.data.success,
        );
        return showError(t(failed?.data?.message || '保存失败，请重试'));
      }

      showSuccess(t('保存成功'));
      await props.refresh();
      await refreshRuntime();
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const next = parseOptions(props.options);
    setInputs(next);
    setSavedInputs(structuredClone(next));
    formRef.current?.setValues(next);
    refreshRuntime();
  }, [props.options]);

  const runtimeConfig = runtime?.config || {};
  const runtimeEnabled = !!runtime?.enabled;
  const runtimeDefaultMax =
    Number(runtimeConfig.default_max_concurrency) > 0
      ? Number(runtimeConfig.default_max_concurrency)
      : DEFAULT_INPUTS['channel_concurrency_setting.default_max_concurrency'];
  const runtimeDefaultRpm = Math.max(
    0,
    Number(runtimeConfig.default_rpm_limit) || 0,
  );
  const runtimeRpmWindow =
    Number(runtimeConfig.rpm_window_seconds) > 0
      ? Number(runtimeConfig.rpm_window_seconds)
      : DEFAULT_INPUTS['channel_concurrency_setting.rpm_window_seconds'];
  const runtimeDiffersFromSaved =
    runtime !== null &&
    (runtimeEnabled !== savedInputs['channel_concurrency_setting.enabled'] ||
      runtimeDefaultMax !==
        Number(
          savedInputs['channel_concurrency_setting.default_max_concurrency'],
        ) ||
      runtimeDefaultRpm !==
        Number(savedInputs['channel_concurrency_setting.default_rpm_limit']) ||
      runtimeRpmWindow !==
        Number(savedInputs['channel_concurrency_setting.rpm_window_seconds']));

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(api) => (formRef.current = api)}
        style={{ marginBottom: 15 }}
      >
        <Form.Section text={t('渠道并发控制')}>
          <Banner
            fullMode={false}
            type='info'
            description={t(
              '开启后，渠道最大并发或 RPM 为 0 时会继承对应全局值；全局 RPM 为 0 表示不限制。只有特殊渠道需要单独覆盖。',
            )}
          />

          <Spin spinning={statsLoading}>
            <Space
              wrap
              style={{ marginTop: 12, marginBottom: 12, width: '100%' }}
            >
              <Text strong>{t('运行状态')}</Text>
              <Tag color={runtimeEnabled ? 'green' : 'grey'}>
                {runtimeEnabled ? t('已启用') : t('未启用')}
              </Tag>
              {runtime && (
                <Text type='tertiary'>
                  {t('实际默认并发')}: {runtimeDefaultMax} · {t('实际默认 RPM')}
                  : {runtimeDefaultRpm || t('不限')} / {runtimeRpmWindow}{' '}
                  {t('秒')} · {t('当前等待请求')}:{' '}
                  {runtime.metrics?.current_waiting ?? 0}
                </Text>
              )}
              {runtimeDiffersFromSaved && (
                <Text type='warning'>
                  {t('运行值与页面配置不同，请检查服务端环境变量覆盖')}
                </Text>
              )}
            </Space>
          </Spin>

          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='channel_concurrency_setting.enabled'
                label={t('启用渠道并发控制')}
                checkedText={t('开')}
                uncheckedText={t('关')}
                extraText={t('关闭时不会限制任何渠道的在途请求数')}
                onChange={setField('channel_concurrency_setting.enabled')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='channel_concurrency_setting.default_max_concurrency'
                label={t('全局默认单渠道并发')}
                min={1}
                step={1}
                style={{ width: '100%' }}
                extraText={t('所有未单独设置并发上限的渠道都使用此值')}
                onChange={setField(
                  'channel_concurrency_setting.default_max_concurrency',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='channel_concurrency_setting.default_rpm_limit'
                label={t('全局默认单渠道 RPM')}
                min={0}
                step={1}
                style={{ width: '100%' }}
                extraText={t('每个渠道在统计窗口内的请求上限，0 表示不限制')}
                onChange={setField(
                  'channel_concurrency_setting.default_rpm_limit',
                )}
              />
            </Col>
          </Row>

          <Collapse
            style={{ marginTop: 8, marginBottom: 16 }}
            defaultActiveKey={[]}
          >
            <Collapse.Panel header={t('过载保护高级参数')} itemKey='advanced'>
              <Row gutter={16}>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.rpm_window_seconds'
                    label={t('RPM 统计窗口')}
                    min={1}
                    step={1}
                    suffix={t('秒')}
                    style={{ width: '100%' }}
                    extraText={t('渠道请求频率统计采用的滑动时间窗口')}
                    onChange={setField(
                      'channel_concurrency_setting.rpm_window_seconds',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.wait_timeout_ms'
                    label={t('全部渠道满载等待时间')}
                    min={1}
                    step={100}
                    suffix='ms'
                    style={{ width: '100%' }}
                    extraText={t('超时仍无容量时返回 503')}
                    onChange={setField(
                      'channel_concurrency_setting.wait_timeout_ms',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.max_queue_length'
                    label={t('等待队列上限')}
                    min={1}
                    step={1}
                    style={{ width: '100%' }}
                    extraText={t('超过上限的新请求会立即返回 503')}
                    onChange={setField(
                      'channel_concurrency_setting.max_queue_length',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.poll_interval_ms'
                    label={t('容量探测间隔')}
                    min={10}
                    step={10}
                    suffix='ms'
                    style={{ width: '100%' }}
                    extraText={t('等待期间重新检查可用渠道的频率')}
                    onChange={setField(
                      'channel_concurrency_setting.poll_interval_ms',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.retry_after_seconds'
                    label={t('客户端重试提示')}
                    min={1}
                    step={1}
                    suffix={t('秒')}
                    style={{ width: '100%' }}
                    extraText={t('写入 503 响应的 Retry-After 响应头')}
                    onChange={setField(
                      'channel_concurrency_setting.retry_after_seconds',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.Switch
                    field='channel_concurrency_setting.affinity_sticky_enabled'
                    label={t('优先等待亲和渠道')}
                    checkedText={t('开')}
                    uncheckedText={t('关')}
                    extraText={t('尽量保持会话渠道不变，以提高上游缓存命中率')}
                    onChange={setField(
                      'channel_concurrency_setting.affinity_sticky_enabled',
                    )}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Form.InputNumber
                    field='channel_concurrency_setting.affinity_wait_ms'
                    label={t('亲和渠道等待时间')}
                    min={1}
                    max={
                      inputs['channel_concurrency_setting.wait_timeout_ms'] ||
                      undefined
                    }
                    step={100}
                    suffix='ms'
                    disabled={
                      !inputs[
                        'channel_concurrency_setting.affinity_sticky_enabled'
                      ]
                    }
                    style={{ width: '100%' }}
                    extraText={t('超时后自动切换到其它未满渠道')}
                    onChange={setField(
                      'channel_concurrency_setting.affinity_wait_ms',
                    )}
                  />
                </Col>
              </Row>
            </Collapse.Panel>
          </Collapse>

          <Space wrap>
            <Button icon={<Save size={16} />} onClick={onSubmit}>
              {t('保存渠道并发设置')}
            </Button>
            <Button
              theme='light'
              icon={<RefreshCw size={16} />}
              loading={statsLoading}
              onClick={refreshRuntime}
            >
              {t('刷新运行状态')}
            </Button>
          </Space>
        </Form.Section>
      </Form>
    </Spin>
  );
}
