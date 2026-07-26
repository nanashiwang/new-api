/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Input, Modal, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { API } from '../../../../helpers/apiCore';
import { timestamp2string } from '../../../../helpers';
import { showError, showSuccess } from '../../../../helpers/toast';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { formatContentSafetyCategory } from '../../../../helpers/contentSafety';

const { Text, Title } = Typography;

const getEmailStatusLabel = (status, t) =>
  ({
    pending: t('待发送'),
    sending: t('发送中'),
    sent: t('已发送'),
    failed: t('发送失败'),
    skipped: t('已跳过'),
  })[status] || status;

const ContentSafetyReviewModal = ({
  visible,
  onCancel,
  user,
  onChanged,
  t,
}) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState('');
  const [violations, setViolations] = useState([]);
  const [reviewCases, setReviewCases] = useState([]);
  const [note, setNote] = useState('');
  const [evidenceByViolation, setEvidenceByViolation] = useState({});
  const [evidenceLoading, setEvidenceLoading] = useState(0);

  const load = useCallback(async () => {
    if (!visible || !user?.id) return;
    setLoading(true);
    try {
      const [violationResponse, caseResponse] = await Promise.all([
        API.get('/api/content-safety/violations', {
          params: { user_id: user.id, page: 1, page_size: 20 },
          disableDuplicate: true,
        }),
        API.get('/api/content-safety/review-cases', {
          params: { user_id: user.id, page: 1, page_size: 20 },
          disableDuplicate: true,
        }),
      ]);
      if (!violationResponse?.data?.success)
        throw new Error(
          violationResponse?.data?.message || t('读取风控记录失败'),
        );
      if (!caseResponse?.data?.success)
        throw new Error(caseResponse?.data?.message || t('读取审核单失败'));
      setViolations(violationResponse.data.data?.items || []);
      setReviewCases(caseResponse.data.data?.items || []);
    } catch (error) {
      showError(error?.message || t('读取风控记录失败'));
    } finally {
      setLoading(false);
    }
  }, [t, user?.id, visible]);

  useEffect(() => {
    if (visible) {
      setNote('');
      setEvidenceByViolation({});
      load();
    }
  }, [load, visible]);

  const pendingCase = useMemo(
    () => reviewCases.find((item) => item.status === 'pending'),
    [reviewCases],
  );

  const resolve = async (resolution) => {
    if (!pendingCase?.id) return;
    setSubmitting(resolution);
    try {
      const response = await API.post(
        `/api/content-safety/review-cases/${pendingCase.id}/resolve`,
        { resolution, note },
      );
      if (!response?.data?.success)
        throw new Error(response?.data?.message || t('提交审核结果失败'));
      showSuccess(t('审核结果已保存'));
      await load();
      onChanged?.();
    } catch (error) {
      showError(error?.message || t('提交审核结果失败'));
    } finally {
      setSubmitting('');
    }
  };

  const confirmPermanentDisable = () => {
    if (!note.trim()) {
      showError(t('永久停用前必须填写审核依据'));
      return;
    }
    Modal.confirm({
      title: t('确认永久停用该用户？'),
      content: t(
        '该操作会立即阻止该用户继续使用服务，并记录管理员、时间和审核依据。请确认你已核对多条脱敏证据。',
      ),
      okText: t('确认停用'),
      cancelText: t('取消'),
      okButtonProps: { type: 'danger' },
      onOk: () => resolve('approved_disable'),
    });
  };

  const revealEvidence = async (violationId) => {
    setEvidenceLoading(violationId);
    try {
      const response = await API.get(
        `/api/content-safety/violations/${violationId}/evidence`,
        { disableDuplicate: true },
      );
      if (!response?.data?.success)
        throw new Error(response?.data?.message || t('读取加密证据失败'));
      setEvidenceByViolation((current) => ({
        ...current,
        [violationId]: response.data.data,
      }));
    } catch (error) {
      showError(error?.message || t('读取加密证据失败'));
    } finally {
      setEvidenceLoading(0);
    }
  };

  return (
    <Modal
      title={`${t('内容风控人工复核')} · ${user?.username || '-'}`}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={isMobile ? 'calc(100vw - 24px)' : 820}
      bodyStyle={{ maxHeight: '72vh', overflowY: 'auto' }}
    >
      <Spin spinning={loading}>
        <div className='space-y-4'>
          <div className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-4'>
            <div className='flex flex-wrap items-center gap-2'>
              <Title heading={6} style={{ margin: 0 }}>
                {t('判断边界')}
              </Title>
              <Tag color='orange'>{t('官方拒绝是事实')}</Tag>
              <Tag color='grey'>{t('细分类可能是本地推断')}</Tag>
            </div>
            <Text type='tertiary' className='mt-2 block'>
              {t(
                '系统仅在上游明确拒绝时加密保存最近用户消息和有限上下文，普通日志与列表不含正文。请结合官方错误码、角色归属、规则来源、置信度和多次历史记录审核；单条记录不能直接证明用户主观恶意。',
              )}
            </Text>
          </div>

          {pendingCase ? (
            <div className='rounded-xl border-2 border-red-300 bg-red-50 p-4'>
              <div className='flex flex-wrap items-center justify-between gap-2'>
                <Title heading={6} style={{ margin: 0, color: '#991b1b' }}>
                  {t('待管理员决定')}
                </Title>
                <Tag color='red'>
                  {t('30天内冷静期')} {pendingCase.window_cooldown_count}{' '}
                  {t('次')}
                </Tag>
              </div>
              <Input
                value={note}
                onChange={setNote}
                maxLength={512}
                placeholder={t('填写审核依据或备注（不会保存用户请求正文）')}
                className='mt-3'
              />
              <div className='mt-3 flex flex-wrap gap-2'>
                <Button
                  loading={submitting === 'observing'}
                  onClick={() => resolve('observing')}
                >
                  {t('继续观察，不停用')}
                </Button>
                <Button
                  type='tertiary'
                  loading={submitting === 'dismissed'}
                  onClick={() => resolve('dismissed')}
                >
                  {t('驳回本次审核')}
                </Button>
                <Button
                  type='danger'
                  theme='solid'
                  loading={submitting === 'approved_disable'}
                  onClick={confirmPermanentDisable}
                >
                  {t('审核确认并永久停用')}
                </Button>
              </div>
            </div>
          ) : (
            <div className='rounded-xl border border-[var(--semi-color-border)] p-3'>
              <Text strong>{t('当前没有待处理审核单')}</Text>
              <Text type='tertiary' className='ml-2'>
                {t('历史记录仍保留30天。')}
              </Text>
            </div>
          )}

          <div>
            <Title heading={6}>{t('最近风控事件')}</Title>
            <div className='space-y-3'>
              {violations.length === 0 ? (
                <Text type='tertiary'>{t('暂无记录')}</Text>
              ) : null}
              {violations.map((item) => (
                <article
                  key={item.id}
                  className='rounded-xl border border-[var(--semi-color-border)] p-4'
                >
                  <div className='flex flex-wrap items-center gap-2'>
                    <Tag color='red'>
                      {item.error_type || '-'} / {item.error_code || '-'}
                    </Tag>
                    <Tag
                      color={
                        item.reason_confidence === 'low' ? 'grey' : 'orange'
                      }
                    >
                      {formatContentSafetyCategory(item.fine_category, t)} ·{' '}
                      {item.reason_confidence || '-'}
                    </Tag>
                    <Text type='tertiary'>
                      {timestamp2string(item.created_at)}
                    </Text>
                    <Tag color={item.evidence_available ? 'blue' : 'grey'}>
                      {item.evidence_available
                        ? t('已加密取证')
                        : t('无可用正文证据')}
                    </Tag>
                    {item.email_status ? (
                      <Tag
                        color={
                          item.email_status === 'sent' ? 'green' : 'orange'
                        }
                      >
                        {t('邮件')} ·{' '}
                        {getEmailStatusLabel(item.email_status, t)}
                      </Tag>
                    ) : (
                      <Tag color='grey'>{t('未创建邮件通知')}</Tag>
                    )}
                  </div>
                  <dl className='mt-3 grid gap-2 text-sm md:grid-cols-2'>
                    <div>
                      <Text type='tertiary'>{t('分类来源')}: </Text>
                      {item.reason_source === 'local_rule'
                        ? t('本地规则推断')
                        : item.reason_source || '-'}
                    </div>
                    <div>
                      <Text type='tertiary'>{t('模型 / 渠道')}: </Text>
                      {item.model_name || '-'} / {item.channel_id || '-'}
                    </div>
                    <div>
                      <Text type='tertiary'>{t('10分钟窗口')}: </Text>
                      {item.burst_count || 0}/3
                    </div>
                    <div>
                      <Text type='tertiary'>{t('30天累计')}: </Text>
                      {item.window_count || 0}
                    </div>
                  </dl>
                  {item.official_message ? (
                    <p className='mb-0 mt-3 break-words text-sm'>
                      <Text type='tertiary'>{t('官方脱敏消息')}: </Text>
                      {item.official_message}
                    </p>
                  ) : null}
                  <p className='mb-0 mt-2 break-words text-sm'>
                    <Text type='tertiary'>{t('原因说明')}: </Text>
                    {item.reason_summary || '-'}
                  </p>
                  <div className='mt-3'>
                    <Button
                      size='small'
                      loading={evidenceLoading === item.id}
                      disabled={
                        !item.evidence_available ||
                        item.fine_category === 'child_sexual_content'
                      }
                      onClick={() => revealEvidence(item.id)}
                    >
                      {item.fine_category === 'child_sexual_content'
                        ? t('受限证据，不可直接展示')
                        : item.evidence_available
                          ? t('审计并查看加密证据')
                          : t('没有捕获到可用用户正文')}
                    </Button>
                  </div>
                  {evidenceByViolation[item.id]?.captured_messages ? (
                    <div className='mt-3 space-y-2 rounded-lg border border-orange-200 bg-orange-50 p-3'>
                      <Text strong>{t('角色分离的请求证据')}</Text>
                      {evidenceByViolation[item.id].captured_messages.map(
                        (message, index) => (
                          <div
                            key={`${message.index}-${message.role}-${index}`}
                            className='rounded border border-orange-100 bg-white p-2 text-sm'
                          >
                            <Tag
                              color={message.role === 'user' ? 'red' : 'grey'}
                            >
                              {message.role}
                            </Tag>
                            <pre className='mb-0 mt-2 whitespace-pre-wrap break-words font-sans'>
                              {message.content}
                            </pre>
                          </div>
                        ),
                      )}
                    </div>
                  ) : null}
                </article>
              ))}
            </div>
          </div>
        </div>
      </Spin>
    </Modal>
  );
};

export default ContentSafetyReviewModal;
