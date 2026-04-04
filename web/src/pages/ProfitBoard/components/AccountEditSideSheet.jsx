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

const { Text } = Typography;

const AccountEditSideSheet = ({
  visible,
  onClose,
  accountDraft,
  setAccountDraft,
  saveAccount,
  deleteAccount,
  savingAccount,
  deletingAccountId,
}) => (
  <SideSheet
    visible={visible}
    onCancel={onClose}
    title={accountDraft.id ? '编辑账户' : '新建账户'}
    width={480}
    footer={
      <div className='flex items-center justify-between'>
        <div>
          {accountDraft.id ? (
            <Button
              type='danger'
              theme='light'
              icon={<Trash2 size={14} />}
              loading={deletingAccountId === accountDraft.id}
              onClick={() => deleteAccount(accountDraft.id)}
            >
              删除
            </Button>
          ) : null}
        </div>
        <Button
          theme='solid'
          type='primary'
          icon={<Save size={14} />}
          loading={savingAccount}
          onClick={saveAccount}
        >
          {accountDraft.id ? '保存账户' : '创建账户'}
        </Button>
      </div>
    }
  >
    <div className='grid gap-4 lg:grid-cols-2'>
      <div>
        <Text type='tertiary' size='small' className='mb-1 block'>
          名称
        </Text>
        <Input
          value={accountDraft.name}
          onChange={(value) =>
            setAccountDraft((prev) => ({ ...prev, name: value }))
          }
          placeholder='例如：Claude便宜渠道'
          prefix={<Pencil size={14} />}
        />
      </div>
      <div className='flex items-end'>
        <div className='flex w-full items-center justify-between rounded-lg border border-semi-color-border bg-semi-color-bg-1 px-3 py-2'>
          <Text strong>启用账户</Text>
          <Switch
            checked={accountDraft.enabled !== false}
            onChange={(checked) =>
              setAccountDraft((prev) => ({
                ...prev,
                enabled: checked,
              }))
            }
          />
        </div>
      </div>

      <div>
        <Text type='tertiary' size='small' className='mb-1 block'>
          URL
        </Text>
        <Input
          value={accountDraft.base_url}
          onChange={(value) =>
            setAccountDraft((prev) => ({
              ...prev,
              base_url: value,
            }))
          }
          placeholder='https://your-new-api.example.com'
        />
      </div>
      <div>
        <Text type='tertiary' size='small' className='mb-1 block'>
          用户 ID
        </Text>
        <InputNumber
          min={0}
          value={accountDraft.user_id || 0}
          onChange={(value) =>
            setAccountDraft((prev) => ({
              ...prev,
              user_id: Number(value || 0),
            }))
          }
          style={{ width: '100%' }}
        />
      </div>

      <div>
        <Text type='tertiary' size='small' className='mb-1 block'>
          密钥
        </Text>
        <Input
          value={accountDraft.access_token}
          onChange={(value) =>
            setAccountDraft((prev) => ({
              ...prev,
              access_token: value,
            }))
          }
          mode='password'
          prefix={<KeyRound size={14} />}
          placeholder={
            accountDraft.access_token_masked
              ? '留空则保留当前密钥'
              : '输入上游 access token'
          }
        />
        {accountDraft.access_token_masked ? (
          <Text type='tertiary' size='small' className='mt-1 block'>
            当前密钥: {accountDraft.access_token_masked}
          </Text>
        ) : null}
      </div>
      <div>
        <Text type='tertiary' size='small' className='mb-1 block'>
          低余额提醒线
        </Text>
        <InputNumber
          min={0}
          value={accountDraft.low_balance_threshold_usd || 0}
          onChange={(value) =>
            setAccountDraft((prev) => ({
              ...prev,
              low_balance_threshold_usd: Number(value || 0),
            }))
          }
          placeholder='不填则不提醒'
          style={{ width: '100%' }}
        />
      </div>

      <div className='lg:col-span-2'>
        <Text type='tertiary' size='small' className='mb-1 block'>
          备注
        </Text>
        <Input
          value={accountDraft.remark}
          onChange={(value) =>
            setAccountDraft((prev) => ({
              ...prev,
              remark: value,
            }))
          }
          placeholder='例如：主站、备用、包月账户'
        />
      </div>
    </div>
  </SideSheet>
);

export default AccountEditSideSheet;
