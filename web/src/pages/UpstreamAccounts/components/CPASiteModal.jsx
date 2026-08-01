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
import { useMemo, useRef } from 'react';
import { Banner, Button, Form, Modal } from '@douyinfe/semi-ui';
import { PlugZap } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  isValidCRSPort,
  joinCRSHostPort,
  splitCRSHostPort,
} from './crsDashboard.utils';
import { isCPAEndpointChanged } from './cpaDashboard.utils';

const SCHEME_OPTIONS = [
  { label: 'https', value: 'https' },
  { label: 'http', value: 'http' },
];

export default function CPASiteModal({
  visible,
  site,
  onSave,
  onTest,
  onCancel,
  saving,
  testing,
}) {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const isEditing = Boolean(site);
  const initialValues = useMemo(() => {
    const { host, port } = splitCRSHostPort(site?.host ?? '');
    return {
      name: site?.name ?? '',
      host,
      port: port || '8317',
      scheme: site?.scheme ?? 'https',
      management_key: '',
      management_key_change: false,
    };
  }, [site]);

  const buildPayload = async () => {
    const values = await formApiRef.current?.validate();
    if (!values) return null;
    const payload = {
      ...values,
      host: joinCRSHostPort(values.host, values.port),
    };
    delete payload.port;
    const endpointChanged =
      isEditing && isCPAEndpointChanged(site, payload.scheme, payload.host);
    if (endpointChanged) payload.management_key_change = true;
    if (isEditing && !payload.management_key_change) {
      payload.management_key = '';
    }
    if (!isEditing) delete payload.management_key_change;
    return payload;
  };

  const handleSave = async () => {
    try {
      const payload = await buildPayload();
      if (payload) onSave(payload);
    } catch {
      // Semi renders validation errors inline.
    }
  };

  const handleTest = async () => {
    try {
      const payload = await buildPayload();
      if (payload) {
        await onTest({
          id: site?.id ?? 0,
          host: payload.host,
          scheme: payload.scheme,
          management_key: payload.management_key,
        });
      }
    } catch {
      // Semi renders validation errors inline.
    }
  };

  return (
    <Modal
      title={isEditing ? t('编辑 CPA 服务') : t('人工接入 CPA')}
      visible={visible}
      onOk={handleSave}
      onCancel={onCancel}
      okText={t('保存并同步')}
      cancelText={t('取消')}
      okButtonProps={{ loading: saving, disabled: saving || testing }}
      cancelButtonProps={{ disabled: saving || testing }}
      width={600}
      centered
      style={{ maxWidth: 'calc(100vw - 24px)' }}
      maskClosable={false}
      bodyStyle={{ padding: '16px 24px 8px' }}
    >
      <Form
        key={site?.id ?? 'new'}
        initValues={initialValues}
        labelPosition='top'
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
      >
        {({ values }) => {
          const endpointChanged =
            isEditing &&
            isCPAEndpointChanged(
              site,
              values.scheme,
              joinCRSHostPort(values.host, values.port),
            );
          const showKeyField =
            !isEditing || values.management_key_change || endpointChanged;
          return (
            <>
              <Form.Section text={t('基础信息')}>
                <Form.Input
                  field='name'
                  label={t('显示名称')}
                  placeholder={t('例如 CPA 主服务')}
                />
                <div className='flex flex-col items-stretch gap-2 sm:flex-row sm:items-start'>
                  <div className='w-full sm:w-[100px]'>
                    <Form.Select
                      field='scheme'
                      label={t('协议')}
                      optionList={SCHEME_OPTIONS}
                      style={{ width: '100%' }}
                    />
                  </div>
                  <div className='min-w-0 flex-1'>
                    <Form.Input
                      field='host'
                      label='Host'
                      placeholder='cpa.example.com'
                      extraText={t('仅填写域名或 IP，无需 http(s)://')}
                      rules={[{ required: true, message: t('请填写 Host') }]}
                    />
                  </div>
                  <div className='w-full sm:w-[120px]'>
                    <Form.Input
                      field='port'
                      label={t('端口')}
                      placeholder='8317'
                      rules={[
                        {
                          validator: (rule, value) => {
                            if (!value || isValidCRSPort(value)) {
                              return Promise.resolve();
                            }
                            return Promise.reject(
                              t('端口必须是 1-65535 的整数'),
                            );
                          },
                        },
                      ]}
                    />
                  </div>
                </div>
              </Form.Section>

              <Form.Section text={t('Management API 凭据')}>
                {isEditing ? (
                  <Form.Switch
                    field='management_key_change'
                    label={t('更改 Management Key')}
                    size='small'
                    extraText={t(
                      '默认保留现有密钥；更改地址或开启此项时必须填写新密钥',
                    )}
                  />
                ) : null}
                {showKeyField ? (
                  <Form.Input
                    field='management_key'
                    label='Management Key'
                    type='password'
                    placeholder={t('CLIProxyAPI remote-management secret-key')}
                    rules={[
                      {
                        required: true,
                        message: t('请填写 Management Key'),
                      },
                    ]}
                  />
                ) : null}
                <Banner
                  type='info'
                  description={t(
                    '密钥仅由 new-api 后端加密保存，并只用于读取 /v0/management/auth-files。',
                  )}
                  closeIcon={null}
                />
                <Button
                  className='mt-3'
                  icon={<PlugZap size={15} />}
                  loading={testing}
                  disabled={saving || testing}
                  onClick={handleTest}
                >
                  {t('测试连接')}
                </Button>
              </Form.Section>
            </>
          );
        }}
      </Form>
    </Modal>
  );
}
