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
  Input,
  InputNumber,
  SideSheet,
  Switch,
  Typography,
} from '@douyinfe/semi-ui';
import { KeyRound, Pencil, Save, Trash2 } from 'lucide-react';
import { getUpstreamAccountSuggestedName } from '../utils';

const { Text, Title } = Typography;

const FieldMessage = ({ message, tone = 'muted' }) => {
  if (!message) return null;
  const className =
    tone === 'error' ? 'mt-1 block text-sm text-red-500' : 'mt-1 block';
  return (
    <Text
      type={tone === 'error' ? 'danger' : 'tertiary'}
      size='small'
      className={className}
    >
      {message}
    </Text>
  );
};

const SectionBlock = ({ title, subtitle, children }) => (
  <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
    <div className='mb-4'>
      <Title heading={6} style={{ margin: 0 }}>
        {title}
      </Title>
      {subtitle ? (
        <Text type='tertiary' size='small' className='mt-1 block'>
          {subtitle}
        </Text>
      ) : null}
    </div>
    <div className='grid gap-4 lg:grid-cols-2'>{children}</div>
  </div>
);

const AccountEditSideSheet = ({
  visible,
  onClose,
  accountDraft,
  updateAccountDraftField,
  normalizeAccountDraftBaseUrl,
  touchAccountDraftField,
  accountDraftErrors,
  accountDraftCanSave,
  accountDraftValidation,
  saveAccount,
  deleteAccount,
  savingAccount,
  deletingAccountId,
  t,
}) => {
  const isEditing = !!accountDraft.id;
  const preparedDraft = accountDraftValidation?.prepared || accountDraft;
  const suggestedName = getUpstreamAccountSuggestedName(accountDraft.base_url);
  const showAutoNameHint =
    !isEditing &&
    suggestedName &&
    String(accountDraft.name || '').trim() === suggestedName;
  const urlPreview =
    preparedDraft.base_url &&
    preparedDraft.base_url !== String(accountDraft.base_url || '').trim()
      ? preparedDraft.base_url
      : '';
  const footerHint = accountDraftCanSave
    ? isEditing
      ? t('保存后可继续手动同步余额')
      : t('创建后会自动同步一次余额状态')
    : t(accountDraftValidation?.firstError || '请先补全必填信息');

  return (
    <SideSheet
      visible={visible}
      onCancel={onClose}
      title={isEditing ? t('编辑账户') : t('新建账户')}
      width={520}
      footer={
        <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <Text
            type={accountDraftCanSave ? 'tertiary' : 'danger'}
            size='small'
            className='max-w-[280px] leading-5'
          >
            {footerHint}
          </Text>
          <div className='flex items-center justify-end gap-2'>
            {isEditing ? (
              <Button
                type='danger'
                theme='light'
                icon={<Trash2 size={14} />}
                loading={deletingAccountId === accountDraft.id}
                onClick={() => deleteAccount(accountDraft.id)}
              >
                {t('删除')}
              </Button>
            ) : null}
            <Button
              theme='solid'
              type='primary'
              icon={<Save size={14} />}
              loading={savingAccount}
              disabled={!accountDraftCanSave}
              onClick={saveAccount}
            >
              {isEditing ? t('保存账户') : t('创建账户')}
            </Button>
          </div>
        </div>
      }
    >
      <div className='space-y-4'>
        <SectionBlock
          title={t('连接信息')}
          subtitle={t('先填 URL、用户 ID 和 access token，系统会自动帮你规范输入')}
        >
          <div className='lg:col-span-2'>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {t('URL')}
            </Text>
            <Input
              value={accountDraft.base_url}
              onChange={(value) => updateAccountDraftField('base_url', value)}
              onBlur={normalizeAccountDraftBaseUrl}
              placeholder='https://your-new-api.example.com'
            />
            <FieldMessage
              message={
                accountDraftErrors.base_url ||
                (urlPreview
                  ? t('将自动规范为 {{url}}', { url: urlPreview })
                  : t('支持直接输入域名，离开输入框时会自动补全 https'))
              }
              tone={accountDraftErrors.base_url ? 'error' : 'muted'}
            />
          </div>

          <div>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {t('用户 ID')}
            </Text>
            <InputNumber
              min={1}
              value={accountDraft.user_id || 0}
              onChange={(value) => updateAccountDraftField('user_id', value)}
              onBlur={() => touchAccountDraftField('user_id')}
              style={{ width: '100%' }}
            />
            <FieldMessage
              message={
                accountDraftErrors.user_id ||
                t('填写远端 new-api 的用户 ID，用于读取钱包余额')
              }
              tone={accountDraftErrors.user_id ? 'error' : 'muted'}
            />
          </div>

          <div>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {t('密钥')}
            </Text>
            <Input
              value={accountDraft.access_token}
              onChange={(value) =>
                updateAccountDraftField('access_token', value)
              }
              onBlur={() => touchAccountDraftField('access_token')}
              mode='password'
              prefix={<KeyRound size={14} />}
              placeholder={
                accountDraft.access_token_masked
                  ? t('留空则保留当前密钥')
                  : t('输入上游 access token')
              }
            />
            <FieldMessage
              message={
                accountDraftErrors.access_token ||
                (accountDraft.access_token_masked
                  ? t(
                      '当前密钥: {{token}}。如需更换，直接输入新的 access token',
                      { token: accountDraft.access_token_masked },
                    )
                  : t('创建后会立即尝试同步余额，建议直接填写可用密钥'))
              }
              tone={accountDraftErrors.access_token ? 'error' : 'muted'}
            />
          </div>
        </SectionBlock>

        <SectionBlock
          title={t('账户信息')}
          subtitle={t('名称支持自动生成，你也可以手动覆盖')}
        >
          <div>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {t('名称')}
            </Text>
            <Input
              value={accountDraft.name}
              onChange={(value) => updateAccountDraftField('name', value)}
              onBlur={() => touchAccountDraftField('name')}
              placeholder={t('例如：主站账号 / 包月账号 / 备用账号')}
              prefix={<Pencil size={14} />}
            />
            <FieldMessage
              message={
                accountDraftErrors.name ||
                (showAutoNameHint
                  ? t('已根据 URL 自动命名为 {{name}}，你可以直接改', {
                      name: suggestedName,
                    })
                  : t('建议用能区分来源或用途的名称，后面选账户会更快'))
              }
              tone={accountDraftErrors.name ? 'error' : 'muted'}
            />
          </div>

          <div className='flex items-end'>
            <div className='flex w-full items-center justify-between rounded-xl border border-semi-color-border bg-semi-color-fill-0 px-3 py-3'>
              <div>
                <Text strong>{t('启用账户')}</Text>
                <Text type='tertiary' size='small' className='mt-1 block'>
                  {t('关闭后会保留配置，但不再参与余额判断')}
                </Text>
              </div>
              <Switch
                checked={accountDraft.enabled !== false}
                onChange={(checked) =>
                  updateAccountDraftField('enabled', checked)
                }
              />
            </div>
          </div>

          <div className='lg:col-span-2'>
            <Text type='tertiary' size='small' className='mb-1 block'>
              {t('备注')}
            </Text>
            <Input
              value={accountDraft.remark}
              onChange={(value) => updateAccountDraftField('remark', value)}
              placeholder={t('例如：主站、备用、包月账户')}
            />
            <FieldMessage
              message={t('可选，用来补充用途、来源或成本特征')}
            />
          </div>
        </SectionBlock>
      </div>
    </SideSheet>
  );
};

export default AccountEditSideSheet;
