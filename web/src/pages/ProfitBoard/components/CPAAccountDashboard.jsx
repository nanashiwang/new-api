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
import React from 'react';
import {
  Button,
  Card,
  Empty,
  Input,
  Select,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  CircleAlert,
  Clock3,
  Gauge,
  PlugZap,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
  WalletCards,
} from 'lucide-react';

const { Text, Title } = Typography;

const SUMMARY_ITEMS = [
  { key: 'total', label: '账号总数', icon: WalletCards, tone: 'blue' },
  { key: 'available', label: '可用账号', icon: ShieldCheck, tone: 'green' },
  { key: 'limited', label: '冷却 / 限速', icon: Clock3, tone: 'orange' },
  { key: 'abnormal', label: '异常账号', icon: CircleAlert, tone: 'red' },
];

const TONE_CLASSES = {
  blue: 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300',
  green: 'bg-green-50 text-green-600 dark:bg-green-900/30 dark:text-green-300',
  orange:
    'bg-orange-50 text-orange-600 dark:bg-orange-900/30 dark:text-orange-300',
  red: 'bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-300',
};

function SummaryCard({ item, t }) {
  const Icon = item.icon;
  return (
    <Card bordered bodyStyle={{ padding: 14 }}>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <Text type='tertiary' size='small'>
            {t(item.label)}
          </Text>
          <div className='mt-1 text-2xl font-semibold tabular-nums'>--</div>
          <Text type='tertiary' size='small' className='mt-1 block'>
            {t('等待首次同步')}
          </Text>
        </div>
        <span
          className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-xl ${TONE_CLASSES[item.tone]}`}
          aria-hidden='true'
        >
          <Icon size={18} />
        </span>
      </div>
    </Card>
  );
}

function DisabledFilters({ t }) {
  return (
    <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-[minmax(220px,1.4fr)_repeat(4,minmax(130px,0.7fr))_auto]'>
      <Input
        prefix={<Search size={15} />}
        placeholder={t('搜索账号或远端 ID')}
        disabled
        aria-label={t('搜索账号或远端 ID')}
      />
      <Select
        value='all-services'
        disabled
        className='w-full'
        aria-label={t('全部服务')}
        optionList={[{ value: 'all-services', label: t('全部服务') }]}
      />
      <Select
        value='all-platforms'
        disabled
        className='w-full'
        aria-label={t('全部平台')}
        optionList={[{ value: 'all-platforms', label: t('全部平台') }]}
      />
      <Select
        value='all-statuses'
        disabled
        className='w-full'
        aria-label={t('全部状态')}
        optionList={[{ value: 'all-statuses', label: t('全部状态') }]}
      />
      <Select
        value='risk-first'
        disabled
        className='w-full'
        aria-label={t('排序方式')}
        optionList={[{ value: 'risk-first', label: t('风险最高优先') }]}
      />
      <Button
        icon={<RefreshCw size={15} />}
        disabled
        className='w-full xl:w-auto'
      >
        {t('刷新账号')}
      </Button>
    </div>
  );
}

function AccountTablePlaceholder({ t }) {
  const columns = [
    '账号',
    '来源 / 服务',
    '状态',
    '额度余量',
    '今日使用',
    '恢复状态',
    '最近同步',
    '操作',
  ];

  return (
    <div className='overflow-hidden rounded-lg border border-solid border-[var(--semi-color-border)]'>
      <div className='hidden grid-cols-[1.2fr_1fr_0.8fr_1.3fr_0.8fr_1fr_1fr_0.55fr] gap-3 bg-[var(--semi-color-fill-0)] px-4 py-3 lg:grid'>
        {columns.map((column) => (
          <Text key={column} type='tertiary' size='small' strong>
            {t(column)}
          </Text>
        ))}
      </div>
      <div className='flex min-h-[250px] items-center justify-center px-4 py-10'>
        <Empty
          image={<ServerCog size={52} strokeWidth={1.4} />}
          title={t('暂无 CPA 账号数据')}
          description={t(
            '服务器配置完成后，这里将展示真实账号状态，不会使用模拟数据。',
          )}
        />
      </div>
    </div>
  );
}

export default function CPAAccountDashboard({ t }) {
  return (
    <div className='space-y-3'>
      <Card bordered bodyStyle={{ padding: 0 }} className='overflow-hidden'>
        <div className='relative overflow-hidden bg-gradient-to-br from-sky-50 via-white to-amber-50 px-4 py-5 dark:from-sky-950/35 dark:via-gray-900 dark:to-amber-950/25 sm:px-5'>
          <div
            className='pointer-events-none absolute -right-12 -top-14 h-44 w-44 rounded-full border-[28px] border-solid opacity-70'
            style={{ borderColor: 'rgba(186, 230, 253, 0.42)' }}
            aria-hidden='true'
          />
          <div className='relative flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
            <div className='min-w-0'>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='flex h-10 w-10 items-center justify-center rounded-xl bg-sky-600 text-white shadow-sm'>
                  <ServerCog size={21} />
                </span>
                <div>
                  <Title heading={5}>{t('CPA 账号概览')}</Title>
                  <Text type='tertiary' size='small'>
                    {t('账号状态、额度窗口、今日用量、恢复时间与最近同步结果')}
                  </Text>
                </div>
                <Tag color='amber' size='small'>
                  {t('等待接入')}
                </Tag>
              </div>
            </div>
            <div className='flex max-w-2xl items-start gap-3 rounded-xl border border-solid border-sky-200/80 bg-white/80 p-3 shadow-sm backdrop-blur dark:border-sky-800/60 dark:bg-gray-900/75'>
              <span className='mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-sky-100 text-sky-700 dark:bg-sky-900/50 dark:text-sky-300'>
                <PlugZap size={17} />
              </span>
              <div className='min-w-0'>
                <div className='flex flex-wrap items-center gap-2'>
                  <Text strong>{t('尚未连接 CPA 服务')}</Text>
                  <Tag color='grey' size='small'>
                    {t('未配置服务')}
                  </Tag>
                </div>
                <Text type='tertiary' size='small' className='mt-1 block'>
                  {t(
                    '提供 CPA 服务器地址和 Management API 凭据后，即可同步账号状态与额度。',
                  )}
                </Text>
                <div className='mt-2 flex flex-wrap gap-1.5'>
                  <Tag color='blue' size='small'>
                    Management API
                  </Tag>
                  <Tag color='cyan' size='small'>
                    {t('默认端口 8317')}
                  </Tag>
                  <Tag color='green' size='small'>
                    {t('只读同步')}
                  </Tag>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
        {SUMMARY_ITEMS.map((item) => (
          <SummaryCard key={item.key} item={item} t={t} />
        ))}
      </div>

      <Card bordered bodyStyle={{ padding: 16 }}>
        <div className='mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
          <div>
            <div className='flex flex-wrap items-center gap-2'>
              <Title heading={5}>{t('CPA 账号使用情况')}</Title>
              <Tag color='grey' size='small'>
                {t('尚未同步')}
              </Tag>
            </div>
            <Text type='tertiary' size='small' className='mt-1 block'>
              {t('接入后可查看')}
              {' · '}
              {t('账号状态、额度窗口、今日用量、恢复时间与最近同步结果')}
            </Text>
          </div>
          <div className='flex items-center gap-1.5 text-amber-600 dark:text-amber-300'>
            <Gauge size={16} />
            <Text type='warning' size='small'>
              {t('配置服务器后可刷新')}
            </Text>
          </div>
        </div>
        <DisabledFilters t={t} />
        <div className='mt-4'>
          <AccountTablePlaceholder t={t} />
        </div>
      </Card>
    </div>
  );
}
