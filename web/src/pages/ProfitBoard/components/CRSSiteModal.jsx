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
import { Form, Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  isValidCRSPort,
  joinCRSHostPort,
  splitCRSHostPort,
} from './crsDashboard.utils';

const SCHEME_OPTIONS = [
  { label: 'https', value: 'https' },
  { label: 'http', value: 'http' },
];

export default function CRSSiteModal({
  visible,
  site,
  onOk,
  onCancel,
  saving,
  groupOptions = [],
}) {
  const { t } = useTranslation();
  const isEditing = !!site;
  const formApiRef = useRef(null);

  const initialValues = useMemo(() => {
    const { host, port } = splitCRSHostPort(site?.host ?? '');
    return {
      name: site?.name ?? '',
      host,
      port,
      scheme: site?.scheme ?? 'https',
      group: site?.group ?? '',
      username: site?.username ?? '',
      password: '',
      password_change: false,
    };
  }, [site]);

  const handleOk = async () => {
    if (!formApiRef.current) return;
    try {
      const values = await formApiRef.current.validate();
      const payload = { ...values };
      payload.host = joinCRSHostPort(values.host, values.port);
      delete payload.port;
      if (!isEditing) delete payload.password_change;
      onOk(payload);
    } catch {
      /* validation failed; Semi shows inline errors */
    }
  };

  return (
    <Modal
      title={isEditing ? t('编辑 CRS 站点') : t('新增 CRS 站点')}
      visible={visible}
      onOk={handleOk}
      onCancel={onCancel}
      okButtonProps={{ loading: saving, disabled: saving }}
      cancelButtonProps={{ disabled: saving }}
      okText={isEditing ? t('保存') : t('创建')}
      cancelText={t('取消')}
      width={580}
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
          const showPasswordField = !isEditing || values.password_change;
          return (
            <>
              <Form.Section text={t('基础信息')}>
                <Form.Input
                  field='name'
                  label={t('显示名称')}
                  placeholder={t('给站点起一个好识别的名字')}
                />
                <div className='flex gap-2 items-start'>
                  <Form.Select
                    field='scheme'
                    label={t('协议')}
                    optionList={SCHEME_OPTIONS}
                    style={{ width: 100 }}
                  />
                  <div className='flex-1 min-w-0'>
                    <Form.Input
                      field='host'
                      label='Host'
                      placeholder='crs-example.meta-api.vip'
                      extraText={t('仅填写域名，无需 http(s)://')}
                      rules={[{ required: true, message: t('请填写 Host') }]}
                    />
                  </div>
                  <div style={{ width: 120 }}>
                    <Form.Input
                      field='port'
                      label={t('端口')}
                      placeholder='8443'
                      extraText={t('可选，范围 1-65535')}
                      rules={[
                        {
                          validator: (rule, value) => {
                            if (!value || isValidCRSPort(value)) {
                              return Promise.resolve();
                            }
                            return Promise.reject(t('端口必须是 1-65535 的整数'));
                          },
                        },
                      ]}
                    />
                  </div>
                </div>
                <Form.Select
                  field='group'
                  label={t('分组')}
                  placeholder={t('例如 codex、shared-crs，可直接输入新值')}
                  optionList={groupOptions}
                  allowCreate
                  filter
                  showClear
                  style={{ width: '100%' }}
                />
              </Form.Section>

              <Form.Section text={t('登录凭证')}>
                <Form.Input
                  field='username'
                  label={t('用户名')}
                  placeholder={t('CRS 管理员用户名')}
                  rules={[{ required: true, message: t('请填写用户名') }]}
                />
                {isEditing && (
                  <Form.Switch
                    field='password_change'
                    label={t('更改密码')}
                    size='small'
                    extraText={t('默认不修改密码；打开后会用新密码替换')}
                  />
                )}
                {showPasswordField && (
                  <Form.Input
                    field='password'
                    label={t('密码')}
                    type='password'
                    placeholder={
                      isEditing ? t('输入新密码') : t('CRS 管理员密码')
                    }
                    rules={
                      !isEditing
                        ? [{ required: true, message: t('请填写密码') }]
                        : undefined
                    }
                  />
                )}
              </Form.Section>
            </>
          );
        }}
      </Form>
    </Modal>
  );
}
