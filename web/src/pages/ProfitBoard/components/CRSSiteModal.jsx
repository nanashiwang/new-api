import { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Modal,
  Select,
  Switch,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const SCHEME_OPTIONS = [
  { label: 'https', value: 'https' },
  { label: 'http', value: 'http' },
];

export default function CRSSiteModal({ visible, site, onOk, onCancel, saving }) {
  const { t } = useTranslation();
  const isEditing = !!site;

  const [form, setForm] = useState({
    name: '',
    host: '',
    scheme: 'https',
    group: '',
    username: '',
    password: '',
    password_change: false,
  });

  useEffect(() => {
    if (!visible) return;
    if (site) {
      setForm({
        name: site.name ?? '',
        host: site.host ?? '',
        scheme: site.scheme ?? 'https',
        group: site.group ?? '',
        username: site.username ?? '',
        password: '',
        password_change: false,
      });
    } else {
      setForm({
        name: '',
        host: '',
        scheme: 'https',
        group: '',
        username: '',
        password: '',
        password_change: false,
      });
    }
  }, [visible, site]);

  const handleChange = (field, value) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  const handleOk = () => {
    const payload = { ...form };
    if (isEditing) {
      payload.password_change = form.password_change;
    }
    onOk(payload);
  };

  const showPasswordField = !isEditing || form.password_change;

  return (
    <Modal
      title={isEditing ? t('编辑 CRS 站点') : t('新增 CRS 站点')}
      visible={visible}
      onOk={handleOk}
      onCancel={onCancel}
      okButtonProps={{ loading: saving, disabled: saving }}
      cancelButtonProps={{ disabled: saving }}
      okText={t('保存')}
      cancelText={t('取消')}
      width={520}
      maskClosable={false}
    >
      <Form
        labelPosition='left'
        labelWidth={100}
        style={{ padding: '8px 0' }}
        onSubmit={handleOk}
      >
        <Form.Input
          field='name'
          label={t('显示名称')}
          placeholder={t('可选，便于识别的名称')}
          value={form.name}
          onChange={(v) => handleChange('name', v)}
        />
        <div className='flex gap-2 items-start'>
          <div style={{ width: 90 }} className='flex-shrink-0'>
            <Form.Select
              field='scheme'
              label={t('协议')}
              value={form.scheme}
              onChange={(v) => handleChange('scheme', v)}
              optionList={SCHEME_OPTIONS}
              style={{ width: 90 }}
            />
          </div>
          <div className='flex-1 min-w-0'>
            <Form.Input
              field='host'
              label={t('Host')}
              placeholder='crs-example.meta-api.vip'
              value={form.host}
              onChange={(v) => handleChange('host', v)}
              required
            />
          </div>
        </div>
        <Form.Input
          field='group'
          label={t('分组')}
          placeholder={t('如 codex、crs-pro-max（可选）')}
          value={form.group}
          onChange={(v) => handleChange('group', v)}
        />
        <Form.Input
          field='username'
          label={t('用户名')}
          placeholder={t('CRS 管理员用户名')}
          value={form.username}
          onChange={(v) => handleChange('username', v)}
          required
        />
        {isEditing && (
          <Form.Slot label={t('更改密码')}>
            <Switch
              checked={form.password_change}
              onChange={(v) => handleChange('password_change', v)}
              size='small'
            />
          </Form.Slot>
        )}
        {showPasswordField && (
          <Form.Input
            field='password'
            label={t('密码')}
            type='password'
            placeholder={isEditing ? t('输入新密码') : t('CRS 管理员密码')}
            value={form.password}
            onChange={(v) => handleChange('password', v)}
            required={!isEditing}
          />
        )}
      </Form>
    </Modal>
  );
}
