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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Banner,
  Button,
  Card,
  Collapsible,
  Empty,
  Spin,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { AlertTriangle, CheckCircle2, RefreshCw } from 'lucide-react';
import { API } from '../../helpers/apiCore';

const { Text, Title } = Typography;

const formatInteger = (value) =>
  new Intl.NumberFormat().format(Number.isFinite(Number(value)) ? value : 0);

const formatDateTime = (value) => {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
};

// 10000 bps 是 1.0 倍，直接展示 bps 让运营核对配置时不必换算两次。
const formatMultiplier = (bps) => {
  const value = Number(bps);
  if (!Number.isFinite(value)) return '-';
  return `${(value / 10000).toFixed(2)}x (${value} bps)`;
};

const formatLag = (seconds, t) => {
  const value = Number(seconds);
  if (!Number.isFinite(value) || value <= 0) return t('无滞后');
  if (value < 60) return t('{{n}} 秒', { n: Math.round(value) });
  if (value < 3600) return t('{{n}} 分钟', { n: Math.round(value / 60) });
  if (value < 86400) return t('{{n}} 小时', { n: Math.round(value / 3600) });
  return t('{{n}} 天', { n: Math.round(value / 86400) });
};

const periodStatusColor = (status) => {
  if (status === 'active') return 'green';
  if (status === 'draft') return 'blue';
  if (status === 'settling') return 'amber';
  return 'grey';
};

// 滞后超过一小时说明摄入已经落后于真实调用，不再只是抖动。
const LAG_WARNING_SECONDS = 3600;

const RuleTable = ({ rules, t }) => {
  const columns = useMemo(
    () => [
      { title: t('规则键'), dataIndex: 'rule_key' },
      { title: t('优先级'), dataIndex: 'priority' },
      {
        title: t('模型匹配'),
        dataIndex: 'model_pattern',
        render: (value) => value || t('全部'),
      },
      {
        title: t('渠道'),
        dataIndex: 'channel_id',
        render: (value) =>
          value === undefined || value === null ? t('全部') : value,
      },
      {
        title: t('计入'),
        dataIndex: 'eligible',
        render: (value) => (
          <Tag color={value ? 'green' : 'grey'}>
            {value ? t('是') : t('否')}
          </Tag>
        ),
      },
      {
        title: t('倍率'),
        dataIndex: 'multiplier_bps',
        render: formatMultiplier,
      },
      { title: t('配置版本'), dataIndex: 'config_version' },
    ],
    [t],
  );

  return (
    <Table
      rowKey='rule_key'
      columns={columns}
      dataSource={Array.isArray(rules) ? rules : []}
      pagination={false}
      size='small'
      empty={
        <Empty description={t('该周期未冻结任何经济规则，事件将全部记为不计入')} />
      }
      scroll={{ x: 760 }}
    />
  );
};

const PeriodCard = ({ period, t }) => {
  const [rulesOpen, setRulesOpen] = useState(period.status === 'active');
  const hasRules = Number(period.rule_count) > 0;

  return (
    <Card className='!rounded-2xl border-0 shadow-sm'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <div className='flex items-center gap-2'>
            <span className='text-lg font-medium'>{period.key}</span>
            <Tag color={periodStatusColor(period.status)}>{period.status}</Tag>
          </div>
          <div className='mt-1 text-sm text-[var(--semi-color-text-2)]'>
            {formatDateTime(period.starts_at)} — {formatDateTime(period.ends_at)}
            <span className='ml-2'>({period.timezone})</span>
          </div>
          <div className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
            {t('配置版本')} {period.config_version || '-'} · {t('随机版本')}{' '}
            {period.random_version || '-'}
          </div>
        </div>
        <Button
          theme='borderless'
          size='small'
          onClick={() => setRulesOpen((open) => !open)}
        >
          {rulesOpen ? t('收起规则') : t('查看规则')}
        </Button>
      </div>

      <div className='mt-4 grid grid-cols-2 gap-3 sm:grid-cols-5'>
        {[
          { label: t('规则数'), value: period.rule_count },
          { label: t('参与用户'), value: period.user_count },
          { label: t('用量事件'), value: period.usage_event_count },
          { label: t('已发放券'), value: period.entitled_tickets },
          { label: t('已消耗券'), value: period.spent_tickets },
        ].map((item) => (
          <div key={item.label}>
            <Text type='tertiary' size='small'>
              {item.label}
            </Text>
            <div className='mt-1 text-xl font-semibold'>
              {formatInteger(item.value)}
            </div>
          </div>
        ))}
      </div>

      {!hasRules && period.status !== 'closed' ? (
        <Banner
          type='warning'
          className='mt-4 !rounded-xl'
          description={t(
            '该周期没有经济规则。周期一旦 active 就不允许原地改规则，只能关闭后新建。',
          )}
        />
      ) : null}

      <Collapsible isOpen={rulesOpen}>
        <div className='mt-4'>
          <RuleTable rules={period.rules} t={t} />
        </div>
      </Collapsible>
    </Card>
  );
};

const PulseOps = () => {
  const { t } = useTranslation();
  const [overview, setOverview] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reloadKey, setReloadKey] = useState(0);

  const retry = useCallback(() => setReloadKey((value) => value + 1), []);

  useEffect(() => {
    const controller = new AbortController();
    let active = true;

    const load = async () => {
      setLoading(true);
      setError('');
      try {
        const response = await API.get('/api/pulse/ops/overview', {
          signal: controller.signal,
          disableDuplicate: true,
          skipErrorHandler: true,
          skipGlobalLoading: true,
        });
        if (!active) return;
        setOverview(response.data || null);
      } catch (requestError) {
        if (!active || requestError?.code === 'ERR_CANCELED') return;
        const status = requestError?.response?.status;
        setError(
          status === 503
            ? t('Pulse 服务暂不可用，请稍后重试')
            : status === 403
              ? t('当前账号无权查看运营概览')
              : t('无法加载运营概览，请稍后重试'),
        );
      } finally {
        if (active) setLoading(false);
      }
    };

    load();
    return () => {
      active = false;
      controller.abort();
    };
  }, [reloadKey, t]);

  const cursorColumns = useMemo(
    () => [
      { title: t('游标'), dataIndex: 'name' },
      { title: t('来源系统'), dataIndex: 'source_system' },
      { title: t('位置'), dataIndex: 'value' },
      {
        title: t('水位时间'),
        dataIndex: 'watermark_at',
        render: formatDateTime,
      },
      {
        title: t('滞后'),
        dataIndex: 'lag_seconds',
        render: (value) => (
          <Tag color={Number(value) > LAG_WARNING_SECONDS ? 'red' : 'green'}>
            {formatLag(value, t)}
          </Tag>
        ),
      },
      { title: t('版本'), dataIndex: 'version' },
    ],
    [t],
  );

  const health = overview?.health;
  const snapshot = overview?.snapshot;

  return (
    <div className='px-2 pb-8'>
      <div className='mx-auto w-full max-w-[1440px]'>
        <div className='mb-5 flex flex-wrap items-center justify-between gap-3'>
          <div>
            <Title heading={3} className='!mb-1'>
              {t('Pulse 运营概览')}
            </Title>
            <Text type='secondary'>
              {t('周期、经济规则与日志摄入进度的只读视图')}
            </Text>
          </div>
          <Button icon={<RefreshCw size={16} />} onClick={retry}>
            {t('刷新')}
          </Button>
        </div>

        {loading ? (
          <Card className='!rounded-2xl border-0 shadow-sm'>
            <div className='flex min-h-[360px] items-center justify-center'>
              <Spin size='large' tip={t('正在加载运营概览')} />
            </div>
          </Card>
        ) : error ? (
          <Card className='!rounded-2xl border-0 shadow-sm'>
            <Empty
              title={t('暂时无法显示运营概览')}
              description={error}
              style={{ padding: '64px 0' }}
            >
              <Button
                type='primary'
                icon={<RefreshCw size={16} />}
                onClick={retry}
              >
                {t('重新加载')}
              </Button>
            </Empty>
          </Card>
        ) : (
          <>
            <Banner
              type={health?.ingest_blocked ? 'danger' : 'success'}
              className='mb-5 !rounded-xl'
              icon={
                health?.ingest_blocked ? (
                  <AlertTriangle size={18} />
                ) : (
                  <CheckCircle2 size={18} />
                )
              }
              title={
                health?.ingest_blocked
                  ? t('用量摄入已停摆')
                  : t('用量摄入正常')
              }
              description={
                health?.ingest_blocked
                  ? health?.blocked_reason ||
                    t('没有可用的活动周期，摄入无法记账')
                  : t('存在覆盖当前时间的活动周期，事件可以正常入账')
              }
            />

            {snapshot ? (
              <div className='mb-5 grid grid-cols-2 gap-4 sm:grid-cols-3 xl:grid-cols-5'>
                {[
                  {
                    label: t('摄入滞后'),
                    value: formatLag(snapshot.ingest_lag_seconds, t),
                  },
                  {
                    label: t('未决冲突'),
                    value: formatInteger(snapshot.open_conflict_count),
                  },
                  {
                    label: t('账本不一致'),
                    value: formatInteger(snapshot.ledger_mismatch_count),
                  },
                  {
                    label: t('结算重试'),
                    value: formatInteger(snapshot.settlement_retry_count),
                  },
                  {
                    label: t('结算死信'),
                    value: formatInteger(snapshot.settlement_dead_count),
                  },
                ].map((item) => (
                  <Card
                    key={item.label}
                    className='!rounded-2xl border-0 shadow-sm'
                  >
                    <Text type='tertiary' size='small'>
                      {item.label}
                    </Text>
                    <div className='mt-2 text-2xl font-semibold'>
                      {item.value}
                    </div>
                  </Card>
                ))}
              </div>
            ) : null}

            <Card
              className='mb-5 !rounded-2xl border-0 shadow-sm'
              title={t('日志摄入游标')}
            >
              <Table
                rowKey='name'
                columns={cursorColumns}
                dataSource={
                  Array.isArray(overview?.cursors) ? overview.cursors : []
                }
                pagination={false}
                size='small'
                empty={<Empty description={t('暂无游标记录')} />}
                scroll={{ x: 760 }}
              />
            </Card>

            <div className='mb-2 flex items-center justify-between'>
              <Title heading={5} className='!mb-0'>
                {t('周期')}
              </Title>
              <Text type='tertiary' size='small'>
                {t('观测时间')} {formatDateTime(overview?.observed_at)}
              </Text>
            </div>

            {Array.isArray(overview?.periods) && overview.periods.length > 0 ? (
              <div className='flex flex-col gap-4'>
                {overview.periods.map((period) => (
                  <PeriodCard key={period.id} period={period} t={t} />
                ))}
              </div>
            ) : (
              <Card className='!rounded-2xl border-0 shadow-sm'>
                <Empty
                  title={t('尚未创建任何周期')}
                  description={t(
                    '周期由运维命令 period-create 创建，需要指定经济规则与倍率。',
                  )}
                  style={{ padding: '48px 0' }}
                />
              </Card>
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default PulseOps;
