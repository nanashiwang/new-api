import React from 'react';
import {
  Banner,
  Button,
  Card,
  Space,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { RefreshCw, Save, Wallet } from 'lucide-react';

const { Text, Title } = Typography;

const ProfitBoardHeader = ({
  querying,
  overviewQuerying,
  runFullRefresh,
  saving,
  saveConfig,
  autoRefreshMode,
  setAutoRefreshMode,
  statusSummary,
  hasNewActivity,
  generatedAtText,
  sharedSiteModelCount,
  warningSummary,
  combinedWarnings,
  sitePriceFactorNote,
  walletModeEnabled,
  selectedAccount,
  t,
}) => (
  <>
    <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
      <div>
        <Title heading={4} style={{ marginBottom: 0 }}>
          {t('收益看板')}
        </Title>
      </div>
      <Space wrap>
        <Button
          theme='solid'
          type='primary'
          icon={<RefreshCw size={16} />}
          loading={querying || overviewQuerying}
          onClick={runFullRefresh}
        >
          {t('刷新收益看板')}
        </Button>
        <Button
          theme='solid'
          type='tertiary'
          icon={<Save size={16} />}
          loading={saving}
          onClick={saveConfig}
        >
          {t('保存配置')}
        </Button>
        <div className='flex items-center gap-2 rounded-full border border-semi-color-border bg-semi-color-fill-0 px-3 py-2'>
          <Text type='tertiary'>{t('低频检查')}</Text>
          <Switch checked={autoRefreshMode} onChange={setAutoRefreshMode} />
        </div>
      </Space>
    </div>

    <Card bordered={false} bodyStyle={{ padding: 16 }}>
      <div
        className={`grid gap-3 ${
          walletModeEnabled
            ? 'lg:grid-cols-[1.2fr_1fr_1fr_1fr_1fr]'
            : 'lg:grid-cols-[1.4fr_1fr_1fr_1fr]'
        }`}
      >
        <div className='rounded-xl bg-semi-color-fill-0 px-4 py-3'>
          <Text type='tertiary'>{t('状态')}</Text>
          <div className='mt-2 flex flex-wrap gap-2'>
            {statusSummary.length > 0 ? (
              statusSummary.map((item) => (
                <Tag key={item.key} color={item.color}>
                  {item.text}
                </Tag>
              ))
            ) : (
              <Tag color='grey'>{t('等待首次刷新')}</Tag>
            )}
            {hasNewActivity ? (
              <Tag color='orange'>{t('检测到新数据，请手动刷新')}</Tag>
            ) : null}
          </div>
        </div>
        <div className='rounded-xl bg-semi-color-fill-0 px-4 py-3'>
          <Text type='tertiary'>{t('上次时间分析')}</Text>
          <div className='mt-2 text-base font-semibold'>{generatedAtText}</div>
        </div>
        <div className='rounded-xl bg-semi-color-fill-0 px-4 py-3'>
          <Text type='tertiary'>{t('问题摘要')}</Text>
          <div className='mt-2 text-base font-semibold'>{warningSummary}</div>
        </div>
        <div className='rounded-xl bg-semi-color-fill-0 px-4 py-3'>
          <Text type='tertiary'>{t('共享本站模型')}</Text>
          <div className='mt-2 text-base font-semibold'>
            {sharedSiteModelCount > 0
              ? `${sharedSiteModelCount} ${t('个模型')}`
              : t('未启用')}
          </div>
        </div>
        {walletModeEnabled ? (
          <div className='rounded-xl bg-semi-color-fill-0 px-4 py-3'>
            <div className='flex items-center gap-2'>
              <Wallet size={15} />
              <Text type='tertiary'>{t('上游钱包')}</Text>
            </div>
            <div className='mt-2 text-sm font-semibold'>
              {selectedAccount?.name || t('未选择账户')}
            </div>
            <div className='mt-2 flex flex-wrap gap-2'>
              {selectedAccount?.status ? (
                <Tag
                  color={
                    {
                      ready: 'green',
                      needs_baseline: 'orange',
                      failed: 'red',
                      disabled: 'grey',
                      not_configured: 'grey',
                    }[selectedAccount.status] || 'grey'
                  }
                >
                  {selectedAccount.status}
                </Tag>
              ) : (
                <Tag color='orange'>{t('等待绑定')}</Tag>
              )}
            </div>
          </div>
        ) : null}
      </div>

      {combinedWarnings.length > 0 || sitePriceFactorNote ? (
        <div className='mt-3 space-y-2'>
          {combinedWarnings.map((warning) => (
            <Banner
              key={warning}
              type='warning'
              description={warning}
              closeIcon={null}
            />
          ))}
          {sitePriceFactorNote ? (
            <Banner
              type='info'
              description={sitePriceFactorNote}
              closeIcon={null}
            />
          ) : null}
        </div>
      ) : null}
    </Card>
  </>
);

export default ProfitBoardHeader;
