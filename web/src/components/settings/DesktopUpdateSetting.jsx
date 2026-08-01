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
  Card,
  Input,
  InputNumber,
  Modal,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  ExternalLink,
  FileJson,
  Github,
  HardDrive,
  KeyRound,
  PackageOpen,
  RefreshCw,
  Trash2,
  UploadCloud,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text, Title } = Typography;

const defaultSettings = {
  enabled: false,
  public_base_url: '',
  max_upload_mb: 256,
  retention_count: 10,
};

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

const DesktopUpdateSetting = () => {
  const { t } = useTranslation();
  const artifactInputRef = useRef(null);
  const manifestInputRef = useRef(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [rotating, setRotating] = useState(false);
  const [status, setStatus] = useState(null);
  const [releases, setReleases] = useState([]);
  const [settings, setSettings] = useState(defaultSettings);
  const [releaseVersion, setReleaseVersion] = useState('');
  const [newToken, setNewToken] = useState('');

  const publicBaseURL = settings.public_base_url.trim().replace(/\/+$/, '');
  const manifestURL = publicBaseURL ? `${publicBaseURL}/latest.json` : '';
  const publishURL = publicBaseURL ? `${publicBaseURL}/publish` : '';

  const refresh = async () => {
    setLoading(true);
    try {
      const [statusResponse, releasesResponse] = await Promise.all([
        API.get('/api/desktop-update/status'),
        API.get('/api/desktop-update/releases'),
      ]);
      if (!statusResponse.data.success) {
        throw new Error(statusResponse.data.message);
      }
      if (!releasesResponse.data.success) {
        throw new Error(releasesResponse.data.message);
      }
      const nextStatus = statusResponse.data.data;
      setStatus(nextStatus);
      setSettings({ ...defaultSettings, ...nextStatus.settings });
      setReleases(releasesResponse.data.data || []);
      if (!releaseVersion && nextStatus.manifest?.version) {
        setReleaseVersion(nextStatus.manifest.version);
      }
    } catch (error) {
      showError(
        error?.response?.data?.message || error.message || t('刷新失败'),
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const saveSettings = async () => {
    setSaving(true);
    try {
      const response = await API.put('/api/desktop-update/settings', settings);
      if (!response.data.success) throw new Error(response.data.message);
      showSuccess(t('保存成功'));
      await refresh();
    } catch (error) {
      showError(
        error?.response?.data?.message || error.message || t('保存失败'),
      );
    } finally {
      setSaving(false);
    }
  };

  const rotateToken = async () => {
    setRotating(true);
    try {
      const response = await API.post('/api/desktop-update/token/rotate');
      if (!response.data.success) throw new Error(response.data.message);
      setNewToken(response.data.data.token);
      showSuccess(t('发布令牌已生成'));
      await refresh();
    } catch (error) {
      showError(
        error?.response?.data?.message || error.message || t('生成失败'),
      );
    } finally {
      setRotating(false);
    }
  };

  const copyText = async (value) => {
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      showSuccess(t('已复制到剪贴板'));
    } catch {
      showError(t('复制失败'));
    }
  };

  const uploadArtifacts = async (event) => {
    const files = Array.from(event.target.files || []);
    event.target.value = '';
    if (!files.length) return;
    const version = releaseVersion.trim().replace(/^v/, '');
    if (!version) {
      showError(t('请先填写版本号'));
      return;
    }
    setUploading(true);
    try {
      for (const file of files) {
        const formData = new FormData();
        formData.append('file', file);
        const response = await API.post(
          `/api/desktop-update/releases/${encodeURIComponent(version)}/files`,
          formData,
        );
        if (!response.data.success) throw new Error(response.data.message);
      }
      showSuccess(t('更新文件上传成功'));
      await refresh();
    } catch (error) {
      showError(
        error?.response?.data?.message || error.message || t('上传失败'),
      );
    } finally {
      setUploading(false);
    }
  };

  const publishManifest = async (event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    setPublishing(true);
    try {
      const response = await API.post('/api/desktop-update/manifest', file, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!response.data.success) throw new Error(response.data.message);
      showSuccess(t('版本发布成功'));
      await refresh();
    } catch (error) {
      showError(
        error?.response?.data?.message || error.message || t('发布失败'),
      );
    } finally {
      setPublishing(false);
    }
  };

  const deleteRelease = (version) => {
    Modal.confirm({
      title: t('删除历史版本'),
      content: t('删除后安装包无法恢复，确定继续吗？'),
      okText: t('删除'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        try {
          const response = await API.delete(
            `/api/desktop-update/releases/${encodeURIComponent(version)}`,
          );
          if (!response.data.success) throw new Error(response.data.message);
          showSuccess(t('删除成功'));
          await refresh();
        } catch (error) {
          showError(
            error?.response?.data?.message || error.message || t('删除失败'),
          );
          throw error;
        }
      },
    });
  };

  const columns = [
    {
      title: t('版本'),
      dataIndex: 'version',
      render: (value, record) => (
        <Space>
          <Text strong>{value}</Text>
          {record.current && <Tag color='green'>{t('当前版本')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('文件'),
      dataIndex: 'files',
      render: (files) => (
        <Space wrap>
          {(files || []).map((file) => (
            <Tag
              key={file.name}
              color='blue'
              onClick={() => file.url && copyText(file.url)}
              style={{ cursor: file.url ? 'pointer' : 'default' }}
            >
              {file.name} · {formatBytes(file.size)}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: t('更新时间'),
      dataIndex: 'modified_at',
      width: 190,
      render: formatDate,
    },
    {
      title: t('操作'),
      width: 100,
      render: (_, record) => (
        <Button
          type='danger'
          theme='borderless'
          icon={<Trash2 size={16} />}
          disabled={record.current}
          onClick={() => deleteRelease(record.version)}
        >
          {t('删除')}
        </Button>
      ),
    },
  ];

  const storageHealthy = status?.storage?.exists && status?.storage?.writable;
  const tokenConfigured = Boolean(status?.token_configured);

  return (
    <Spin spinning={loading} size='large'>
      <div className='mt-3 flex flex-col gap-4'>
        <Card>
          <div className='flex flex-col gap-4'>
            <div className='flex flex-wrap items-start justify-between gap-3'>
              <div>
                <Title heading={4} style={{ margin: 0 }}>
                  {t('桌面客户端更新')}
                </Title>
                <Text type='tertiary'>
                  {t(
                    '由 NewAPI 统一托管更新清单和安装包，无需增加 Nginx 路由。',
                  )}
                </Text>
              </div>
              <Button
                icon={<RefreshCw size={16} />}
                onClick={refresh}
                loading={loading}
              >
                {t('刷新状态')}
              </Button>
            </div>

            <div className='grid grid-cols-1 gap-3 md:grid-cols-3'>
              <Card shadows='hover'>
                <Space align='start'>
                  <HardDrive size={20} />
                  <div>
                    <Text strong>{t('存储目录')}</Text>
                    <div className='break-all text-xs text-gray-500'>
                      {status?.storage?.directory || '-'}
                    </div>
                    <Tag color={storageHealthy ? 'green' : 'orange'}>
                      {storageHealthy ? t('可写') : t('不可用')}
                    </Tag>
                  </div>
                </Space>
              </Card>
              <Card shadows='hover'>
                <Space align='start'>
                  <PackageOpen size={20} />
                  <div>
                    <Text strong>{t('当前发布版本')}</Text>
                    <div className='text-lg font-semibold'>
                      {status?.manifest?.version || t('尚未发布')}
                    </div>
                    <Text type='tertiary'>
                      {(status?.manifest?.platforms || []).join(' · ') || '-'}
                    </Text>
                  </div>
                </Space>
              </Card>
              <Card shadows='hover'>
                <Space align='start'>
                  <KeyRound size={20} />
                  <div>
                    <Text strong>{t('自动发布令牌')}</Text>
                    <div>
                      <Tag color={tokenConfigured ? 'green' : 'orange'}>
                        {tokenConfigured ? t('已配置') : t('未配置')}
                      </Tag>
                    </div>
                    <Text type='tertiary'>
                      {status?.token_source === 'environment'
                        ? t('来自环境变量')
                        : status?.token_source === 'database'
                          ? t('由管理后台维护')
                          : '-'}
                    </Text>
                  </div>
                </Space>
              </Card>
            </div>

            {status?.manifest_error && (
              <Banner
                type='warning'
                icon={<AlertTriangle size={18} />}
                description={status.manifest_error}
              />
            )}
          </div>
        </Card>

        <Card title={t('服务配置')}>
          <div className='grid grid-cols-1 gap-5 lg:grid-cols-2'>
            <div className='flex flex-col gap-2'>
              <Text strong>{t('启用桌面更新服务')}</Text>
              <Switch
                checked={settings.enabled}
                onChange={(enabled) =>
                  setSettings((current) => ({ ...current, enabled }))
                }
              />
              <Text type='tertiary'>
                {t('关闭后公开清单和安装包返回 404，但管理和预上传仍可使用。')}
              </Text>
            </div>
            <div className='flex flex-col gap-2'>
              <Text strong>{t('对外基础地址')}</Text>
              <Input
                value={settings.public_base_url}
                placeholder='https://example.com/desktop/update'
                onChange={(public_base_url) =>
                  setSettings((current) => ({
                    ...current,
                    public_base_url,
                  }))
                }
              />
              <Text type='tertiary'>{manifestURL || '-'}</Text>
            </div>
            <div className='flex flex-col gap-2'>
              <Text strong>{t('单文件上传限制')}</Text>
              <InputNumber
                min={1}
                max={4096}
                suffix='MB'
                value={settings.max_upload_mb}
                onChange={(max_upload_mb) =>
                  setSettings((current) => ({
                    ...current,
                    max_upload_mb: Number(max_upload_mb) || 1,
                  }))
                }
              />
            </div>
            <div className='flex flex-col gap-2'>
              <Text strong>{t('保留版本数量')}</Text>
              <InputNumber
                min={0}
                max={100}
                value={settings.retention_count}
                onChange={(retention_count) =>
                  setSettings((current) => ({
                    ...current,
                    retention_count: Number(retention_count) || 0,
                  }))
                }
              />
              <Text type='tertiary'>{t('设为 0 时不自动清理旧版本。')}</Text>
            </div>
          </div>
          <div className='mt-5 flex flex-wrap gap-2'>
            <Button type='primary' loading={saving} onClick={saveSettings}>
              {t('保存配置')}
            </Button>
            {manifestURL && (
              <Button
                icon={<ExternalLink size={16} />}
                onClick={() => window.open(manifestURL, '_blank', 'noopener')}
              >
                {t('测试公开地址')}
              </Button>
            )}
          </div>
        </Card>

        <Card title={t('发布令牌')}>
          <Banner
            type='warning'
            description={t(
              '令牌只在生成时显示一次。轮换后，GitHub Actions 中的旧令牌会立即失效。',
            )}
            style={{ marginBottom: 16 }}
          />
          <Banner
            type='info'
            description={
              <span>
                {t('GitHub Actions 需要配置两个仓库密钥：')}{' '}
                <Text code>DESKTOP_UPDATE_PUBLISH_URL</Text>
                {' / '}
                <Text code>DESKTOP_UPDATE_PUBLISH_TOKEN</Text>
              </span>
            }
            style={{ marginBottom: 16 }}
          />
          <Space wrap>
            <Button
              icon={<KeyRound size={16} />}
              loading={rotating}
              onClick={rotateToken}
            >
              {tokenConfigured ? t('轮换发布令牌') : t('生成发布令牌')}
            </Button>
            {publishURL && (
              <Button
                icon={<Copy size={16} />}
                onClick={() => copyText(publishURL)}
              >
                {t('复制发布地址')}
              </Button>
            )}
          </Space>
          {newToken && (
            <div className='mt-4 rounded-lg border border-orange-200 bg-orange-50 p-4'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Text strong>{t('请立即保存这个令牌')}</Text>
                <Input
                  value={newToken}
                  readOnly
                  suffix={
                    <Button
                      theme='borderless'
                      icon={<Copy size={16} />}
                      onClick={() => copyText(newToken)}
                    />
                  }
                />
              </Space>
            </div>
          )}
        </Card>

        <Card title={t('手动发布')}>
          <Banner
            type='info'
            description={t(
              '先上传所有签名安装包，最后上传 GitHub Release 生成的 latest.json。服务会校验文件并将下载地址改写为平台地址。',
            )}
            style={{ marginBottom: 16 }}
          />
          <div className='flex flex-col gap-3 md:flex-row md:items-end'>
            <div className='min-w-0 flex-1'>
              <Text strong>{t('发布版本')}</Text>
              <Input
                value={releaseVersion}
                placeholder='0.1.17'
                onChange={setReleaseVersion}
              />
            </div>
            <Button
              icon={<UploadCloud size={16} />}
              loading={uploading}
              onClick={() => artifactInputRef.current?.click()}
            >
              {t('上传更新文件')}
            </Button>
            <Button
              type='primary'
              icon={<FileJson size={16} />}
              loading={publishing}
              onClick={() => manifestInputRef.current?.click()}
            >
              {t('上传并发布 latest.json')}
            </Button>
          </div>
          <input
            ref={artifactInputRef}
            type='file'
            multiple
            className='hidden'
            accept='.gz,.sig,.AppImage,.msi,.exe,.dmg,.zip,.deb,.rpm'
            onChange={uploadArtifacts}
          />
          <input
            ref={manifestInputRef}
            type='file'
            className='hidden'
            accept='.json,application/json'
            onChange={publishManifest}
          />
        </Card>

        <Card
          title={t('历史版本')}
          headerExtraContent={
            <Space>
              <Github size={16} />
              <Text type='tertiary'>
                {t('GitHub 自动发布时也会复用同一套校验流程')}
              </Text>
            </Space>
          }
        >
          <Table
            columns={columns}
            dataSource={releases}
            rowKey='version'
            pagination={false}
            empty={<Text type='tertiary'>{t('暂无桌面更新版本')}</Text>}
          />
        </Card>

        {status?.settings?.enabled && status?.manifest && storageHealthy && (
          <Banner
            type='success'
            icon={<CheckCircle2 size={18} />}
            description={t('桌面更新服务运行正常')}
          />
        )}
      </div>
    </Spin>
  );
};

export default DesktopUpdateSetting;
