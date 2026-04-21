import React from 'react';
import {
  Checkbox,
  Empty,
  Modal,
  Radio,
  RadioGroup,
  Select,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';

const getChannelStatusMeta = (channel, t) => {
  if (channel?.status === 2) {
    return { color: 'red', text: t('已禁用') };
  }
  if (channel?.status === 3) {
    return { color: 'yellow', text: t('自动禁用') };
  }
  if (channel?.effective_available === false) {
    return { color: 'orange', text: t('已启用（暂不可选）') };
  }
  return { color: 'green', text: t('已启用') };
};

const TagTestModal = ({
  visible,
  currentTagTestGroup,
  tagTestMode,
  tagTestScope,
  setTagTestScope,
  tagTestChannels,
  tagTestLoading,
  tagTestSubmitting,
  selectedTagTestChannelIds,
  setSelectedTagTestChannelIds,
  tagTestModelOptions,
  selectedTagTestModel,
  setSelectedTagTestModel,
  onCancel,
  onConfirm,
  t,
}) => {
  const channelCount = tagTestChannels.length;
  const isModelMode = tagTestMode === 'model';
  const okDisabled =
    tagTestLoading ||
    tagTestSubmitting ||
    (channelCount === 0 && !tagTestLoading) ||
    (tagTestScope === 'specified' && selectedTagTestChannelIds.length === 0) ||
    (isModelMode && !selectedTagTestModel);

  return (
    <Modal
      title={
        currentTagTestGroup
          ? `${currentTagTestGroup.name} ${isModelMode ? t('模型测试') : t('测试')}`
          : t('标签测试')
      }
      visible={visible}
      onCancel={onCancel}
      onOk={onConfirm}
      okText={tagTestSubmitting ? t('测试中...') : t('开始测试')}
      cancelText={t('取消')}
      confirmLoading={tagTestSubmitting}
      okButtonProps={{ disabled: okDisabled }}
      size='large'
      centered
    >
      <Spin spinning={tagTestLoading}>
        <div className='flex flex-col gap-4'>
          <div className='flex flex-col gap-2'>
            <Typography.Text strong>{t('测试范围')}</Typography.Text>
            <RadioGroup
              direction='vertical'
              value={tagTestScope}
              onChange={(value) =>
                setTagTestScope(value?.target?.value || value)
              }
            >
              <Radio value='all'>
                {t('测试标签下全部渠道')} ({channelCount})
              </Radio>
              <Radio value='specified'>{t('指定渠道')}</Radio>
            </RadioGroup>
          </div>

          {isModelMode ? (
            <div className='flex flex-col gap-2'>
              <Typography.Text strong>{t('测试模型')}</Typography.Text>
              <Select
                value={selectedTagTestModel}
                onChange={setSelectedTagTestModel}
                optionList={tagTestModelOptions}
                placeholder={t('请选择要测试的模型')}
                filter
                searchPosition='dropdown'
                style={{ width: '100%' }}
              />
            </div>
          ) : null}

          {tagTestScope === 'specified' ? (
            <div className='flex flex-col gap-2'>
              <Typography.Text strong>{t('可选渠道')}</Typography.Text>
              {channelCount === 0 ? (
                <Empty
                  image={
                    <IllustrationNoResult style={{ width: 120, height: 120 }} />
                  }
                  darkModeImage={
                    <IllustrationNoResultDark
                      style={{ width: 120, height: 120 }}
                    />
                  }
                  description={t('暂无可选渠道')}
                  style={{ padding: 24 }}
                />
              ) : (
                <div
                  className='border border-[var(--semi-color-border)] rounded-lg p-3 overflow-y-auto'
                  style={{ maxHeight: 320 }}
                >
                  <Checkbox.Group
                    value={selectedTagTestChannelIds}
                    onChange={(values) => setSelectedTagTestChannelIds(values)}
                  >
                    <div className='flex flex-col gap-2'>
                      {tagTestChannels.map((channel) => {
                        const statusMeta = getChannelStatusMeta(channel, t);
                        return (
                          <Checkbox
                            key={channel.id}
                            value={String(channel.id)}
                            className='!mr-0'
                          >
                            <div className='flex items-center gap-2 flex-wrap'>
                              <span>{channel.name || `#${channel.id}`}</span>
                              <Tag color={statusMeta.color} shape='circle' size='small'>
                                {statusMeta.text}
                              </Tag>
                            </div>
                          </Checkbox>
                        );
                      })}
                    </div>
                  </Checkbox.Group>
                </div>
              )}
            </div>
          ) : null}
        </div>
      </Spin>
    </Modal>
  );
};

export default TagTestModal;
