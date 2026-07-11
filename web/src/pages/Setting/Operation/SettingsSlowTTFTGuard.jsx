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
  Divider,
  Form,
  Modal,
  Row,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';
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
  'slow_ttft_setting.enabled': true,
  'slow_ttft_setting.observe_only': false,
  'slow_ttft_setting.threshold_ms': 8000,
  'slow_ttft_setting.baseline_multiplier': 3,
  'slow_ttft_setting.max_sample_ms': 120000,
  'slow_ttft_setting.baseline_refresh_seconds': 3600,
  'slow_ttft_setting.baseline_min_samples': 6,
  'slow_ttft_setting.baseline_min_peer_tags': 1,
  'slow_ttft_setting.evidence_window_seconds': 900,
  'slow_ttft_setting.global_min_samples': 12,
  'slow_ttft_setting.global_slow_rate': 0.6,
  'slow_ttft_setting.global_min_users': 3,
  'slow_ttft_setting.global_min_traces': 5,
  'slow_ttft_setting.global_circuit_seconds': 300,
  'slow_ttft_setting.trace_consecutive_slow': 3,
  'slow_ttft_setting.trace_circuit_seconds': 1800,
  'slow_ttft_setting.max_entries': 10000,
  'slow_ttft_setting.context_bucket_boundaries': JSON.stringify([
    50000, 100000, 150000, 200000,
  ]),
};

const BOOLEAN_KEYS = new Set([
  'slow_ttft_setting.enabled',
  'slow_ttft_setting.observe_only',
]);

const JSON_KEYS = new Set(['slow_ttft_setting.context_bucket_boundaries']);

const EMPTY_STATS = {
  enabled: false,
  observe_only: false,
  last_baseline_refresh_at: 0,
  next_baseline_refresh_at: 0,
  pending_baseline_entries: 0,
  baseline_entries: 0,
  evidence_entries: 0,
  trace_entries: 0,
  open_global_circuits: 0,
  active_trace_blocks: 0,
  dropped_entries: 0,
  max_entries: 0,
  circuits: [],
};

const formatTimestamp = (seconds) => {
  if (!seconds) return '-';
  return new Date(Number(seconds) * 1000).toLocaleString();
};

const parseBucketBoundaries = (raw) => {
  try {
    const values = JSON.parse(raw || '[]');
    if (!Array.isArray(values) || values.length < 1 || values.length > 10) {
      return null;
    }
    let previous = 0;
    for (const value of values) {
      if (!Number.isInteger(value) || value <= previous || value > 1000000) {
        return null;
      }
      previous = value;
    }
    return values;
  } catch (error) {
    return null;
  }
};

export default function SettingsSlowTTFTGuard(props) {
  const { t } = useTranslation();
  const { Text } = Typography;
  const formRef = useRef();
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [inputs, setInputs] = useState(DEFAULT_INPUTS);
  const [savedInputs, setSavedInputs] = useState(DEFAULT_INPUTS);
  const [stats, setStats] = useState(EMPTY_STATS);

  const setField = (key) => (value) => {
    setInputs((current) => ({ ...current, [key]: value }));
  };

  const refreshStats = async () => {
    try {
      setStatsLoading(true);
      const response = await API.get('/api/option/slow_ttft_guard', {
        disableDuplicate: true,
      });
      const { success, message, data } = response.data;
      if (!success) return showError(t(message));
      setStats({ ...EMPTY_STATS, ...(data || {}) });
    } catch (error) {
      showError(t('刷新慢首字保护状态失败'));
    } finally {
      setStatsLoading(false);
    }
  };

  const refreshBaselines = async () => {
    try {
      setStatsLoading(true);
      const response = await API.post('/api/option/slow_ttft_guard/refresh');
      const { success, message, data } = response.data;
      if (!success) return showError(t(message));
      setStats({ ...EMPTY_STATS, ...(data || {}) });
      showSuccess(t('同行基线已重新计算'));
    } catch (error) {
      showError(t('重新计算同行基线失败'));
    } finally {
      setStatsLoading(false);
    }
  };

  const confirmClearState = () => {
    Modal.confirm({
      title: t('确认清空慢首字保护状态'),
      content: t('将清空当前实例的同行基线、证据窗口和熔断状态。'),
      onOk: async () => {
        try {
          const response = await API.delete('/api/option/slow_ttft_guard');
          const { success, message } = response.data;
          if (!success) return showError(t(message));
          showSuccess(t('已清空'));
          await refreshStats();
        } catch (error) {
          showError(t('清空慢首字保护状态失败'));
        }
      },
    });
  };

  const onSubmit = async () => {
    const changes = compareObjects(inputs, savedInputs);
    if (!changes.length) return showWarning(t('你似乎并没有修改什么'));

    const buckets = parseBucketBoundaries(
      inputs['slow_ttft_setting.context_bucket_boundaries'],
    );
    if (!buckets) {
      return showError(t('上下文分桶必须是 1-10 个严格递增正整数的 JSON 数组'));
    }

    try {
      setLoading(true);
      const responses = await Promise.all(
        changes.map(({ key }) => {
          let value = inputs[key];
          if (JSON_KEYS.has(key)) value = JSON.stringify(buckets);
          else value = String(value ?? '');
          return API.put('/api/option/', { key, value });
        }),
      );
      if (responses.some((response) => !response?.data?.success)) {
        const failed = responses.find(
          (response) => response?.data && !response.data.success,
        );
        return showError(t(failed?.data?.message || '保存失败，请重试'));
      }
      showSuccess(t('保存成功'));
      await props.refresh();
      await refreshStats();
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const next = { ...DEFAULT_INPUTS };
    for (const key of Object.keys(DEFAULT_INPUTS)) {
      if (!(key in props.options)) continue;
      const raw = props.options[key];
      if (BOOLEAN_KEYS.has(key)) next[key] = toBoolean(raw);
      else if (JSON_KEYS.has(key)) {
        try {
          next[key] = JSON.stringify(JSON.parse(raw));
        } catch (error) {
          next[key] = raw;
        }
      } else {
        const value = Number(raw);
        next[key] = Number.isFinite(value) ? value : DEFAULT_INPUTS[key];
      }
    }
    setInputs(next);
    setSavedInputs(structuredClone(next));
    formRef.current?.setValues(next);
    refreshStats();
  }, [props.options]);

  const circuitColumns = [
    { title: t('模型'), dataIndex: 'model' },
    { title: t('分组'), dataIndex: 'group' },
    {
      title: 'Tag',
      dataIndex: 'tag',
      render: (value) => <Tag color='red'>{value || '-'}</Tag>,
    },
    {
      title: t('触发证据'),
      render: (_, row) =>
        `${row.trigger_slow || 0}/${row.trigger_total || 0} (${(
          Number(row.trigger_rate || 0) * 100
        ).toFixed(0)}%)`,
    },
    {
      title: t('用户 / Trace'),
      render: (_, row) =>
        `${row.distinct_users || 0} / ${row.distinct_traces || 0}`,
    },
    {
      title: t('恢复时间'),
      dataIndex: 'open_until',
      render: formatTimestamp,
    },
  ];

  const metricItems = [
    [t('已生成同行基线'), stats.baseline_entries],
    [t('待计算样本项'), stats.pending_baseline_entries],
    [t('全局 Tag 熔断'), stats.open_global_circuits],
    [t('Trace 软屏蔽'), stats.active_trace_blocks],
    [t('Tag 证据项'), stats.evidence_entries],
    [t('Trace 状态项'), stats.trace_entries],
    [t('容量丢弃项'), stats.dropped_entries],
    [t('状态容量上限'), stats.max_entries],
  ];

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(api) => (formRef.current = api)}
        style={{ marginBottom: 15 }}
      >
        <Form.Section text={t('慢首字 Tag 自动换路')}>
          <Banner
            fullMode={false}
            type='warning'
            description={t(
              '仅在首个有效输出同时超过绝对阈值和同行基线倍数时记录为慢请求；当前请求不重试，只影响下一次选路。previous_response_id 请求只观察，不跨 Tag。',
            )}
          />
          <Text type='tertiary' style={{ display: 'block', marginTop: 8 }}>
            {t(
              '请求结束仅更新内存计数，不扫描日志、不逐请求写 Redis 或数据库；同行基线默认每小时计算一次。',
            )}
          </Text>

          <Divider margin='12px' />
          <Row gutter={16}>
            <Col xs={24} sm={12} md={6}>
              <Form.Switch
                field='slow_ttft_setting.enabled'
                label={t('启用慢首字保护')}
                checkedText={t('｜')}
                uncheckedText={t('〇')}
                onChange={setField('slow_ttft_setting.enabled')}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Form.Switch
                field='slow_ttft_setting.observe_only'
                label={t('仅观察，不换路')}
                checkedText={t('｜')}
                uncheckedText={t('〇')}
                onChange={setField('slow_ttft_setting.observe_only')}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Form.InputNumber
                field='slow_ttft_setting.max_entries'
                label={t('内存状态上限')}
                min={100}
                max={100000}
                onChange={setField('slow_ttft_setting.max_entries')}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Form.InputNumber
                field='slow_ttft_setting.max_sample_ms'
                label={t('基线样本最大值（ms）')}
                min={1000}
                max={600000}
                onChange={setField('slow_ttft_setting.max_sample_ms')}
              />
            </Col>
          </Row>

          <Divider align='left'>{t('慢请求判定')}</Divider>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.threshold_ms'
                label={t('绝对阈值（ms）')}
                min={1000}
                max={120000}
                onChange={setField('slow_ttft_setting.threshold_ms')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.baseline_multiplier'
                label={t('同行基线倍数')}
                min={1}
                max={20}
                step={0.1}
                onChange={setField('slow_ttft_setting.baseline_multiplier')}
              />
            </Col>
            <Col xs={24} sm={24} md={8}>
              <Form.Input
                field='slow_ttft_setting.context_bucket_boundaries'
                label={t('上下文分桶边界（tokens JSON）')}
                onChange={setField(
                  'slow_ttft_setting.context_bucket_boundaries',
                )}
              />
            </Col>
          </Row>

          <Divider align='left'>{t('同行基线')}</Divider>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.baseline_refresh_seconds'
                label={t('重算周期（秒）')}
                min={3600}
                max={86400}
                extraText={t('默认 3600 秒，每小时仅计算一次。')}
                onChange={setField(
                  'slow_ttft_setting.baseline_refresh_seconds',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.baseline_min_samples'
                label={t('每个 Tag 最少样本')}
                min={1}
                max={10000}
                onChange={setField('slow_ttft_setting.baseline_min_samples')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.baseline_min_peer_tags'
                label={t('最少同行 Tag 数')}
                min={1}
                max={50}
                onChange={setField('slow_ttft_setting.baseline_min_peer_tags')}
              />
            </Col>
          </Row>

          <Divider align='left'>{t('Trace 级软屏蔽')}</Divider>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.trace_consecutive_slow'
                label={t('连续慢请求次数')}
                min={1}
                max={20}
                onChange={setField('slow_ttft_setting.trace_consecutive_slow')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.trace_circuit_seconds'
                label={t('Trace 屏蔽时长（秒）')}
                min={30}
                max={86400}
                onChange={setField('slow_ttft_setting.trace_circuit_seconds')}
              />
            </Col>
          </Row>

          <Divider align='left'>{t('全局 Tag 熔断')}</Divider>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.evidence_window_seconds'
                label={t('证据窗口（秒）')}
                min={60}
                max={3600}
                onChange={setField('slow_ttft_setting.evidence_window_seconds')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.global_min_samples'
                label={t('最少总样本')}
                min={1}
                max={10000}
                onChange={setField('slow_ttft_setting.global_min_samples')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.global_slow_rate'
                label={t('慢请求比例（0-1）')}
                min={0.01}
                max={1}
                step={0.05}
                onChange={setField('slow_ttft_setting.global_slow_rate')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.global_min_users'
                label={t('最少独立用户数')}
                min={2}
                max={20}
                extraText={t('默认 3，可在 2-20 之间配置。')}
                onChange={setField('slow_ttft_setting.global_min_users')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.global_min_traces'
                label={t('最少独立 Trace 数')}
                min={1}
                max={100}
                extraText={t('Trace 来自已命中的渠道亲和规则。')}
                onChange={setField('slow_ttft_setting.global_min_traces')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='slow_ttft_setting.global_circuit_seconds'
                label={t('Tag 熔断时长（秒）')}
                min={30}
                max={86400}
                onChange={setField('slow_ttft_setting.global_circuit_seconds')}
              />
            </Col>
          </Row>

          <Space style={{ marginTop: 12 }} wrap>
            <Button theme='solid' type='primary' onClick={onSubmit}>
              {t('保存慢首字设置')}
            </Button>
            <Button
              icon={<IconRefresh />}
              loading={statsLoading}
              onClick={refreshStats}
            >
              {t('刷新状态')}
            </Button>
            <Button loading={statsLoading} onClick={refreshBaselines}>
              {t('立即计算同行基线')}
            </Button>
            <Button type='danger' onClick={confirmClearState}>
              {t('清空保护状态')}
            </Button>
          </Space>

          <Divider align='left'>{t('当前实例状态')}</Divider>
          <Row gutter={[12, 12]}>
            {metricItems.map(([label, value]) => (
              <Col xs={12} sm={8} md={6} key={label}>
                <div
                  style={{
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 8,
                    padding: '10px 12px',
                  }}
                >
                  <Text type='tertiary' size='small'>
                    {label}
                  </Text>
                  <div style={{ fontSize: 20, fontWeight: 600 }}>
                    {Number(value || 0)}
                  </div>
                </div>
              </Col>
            ))}
          </Row>
          <Text type='tertiary' style={{ display: 'block', marginTop: 10 }}>
            {t('上次基线计算')}：
            {formatTimestamp(stats.last_baseline_refresh_at)}；
            {t('预计下次计算')}：
            {formatTimestamp(stats.next_baseline_refresh_at)}
          </Text>

          <Table
            style={{ marginTop: 12 }}
            columns={circuitColumns}
            dataSource={stats.circuits || []}
            rowKey={(row) => `${row.model}-${row.group}-${row.tag}`}
            pagination={false}
            size='small'
            empty={t('当前没有全局 Tag 熔断')}
          />
        </Form.Section>
      </Form>
    </Spin>
  );
}
