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

import React, { useMemo, useRef, useState } from 'react';
import {
  Form,
  Button,
  SideSheet,
  Space,
  Select,
  InputNumber,
  Divider,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch, IconFilter } from '@douyinfe/semi-icons';

const UsersFilters = ({
  formInitValues,
  setFormApi,
  searchUsers,
  pageSize,
  groupOptions,
  advancedFilters,
  defaultAdvancedFilters,
  applyAdvancedFilters,
  resetAdvancedFilters,
  loading,
  searching,
  t,
}) => {
  const formApiRef = useRef(null);
  const [advancedVisible, setAdvancedVisible] = useState(false);
  const [draftAdvancedFilters, setDraftAdvancedFilters] = useState(
    defaultAdvancedFilters,
  );
  const { Text } = Typography;

  // 将高级筛选选项集中管理，避免 JSX 字面量分散，便于后续维护。
  const roleOptions = useMemo(
    () => [
      { label: t('全部角色'), value: '' },
      { label: t('普通用户'), value: '1' },
      { label: t('管理员'), value: '10' },
      { label: t('超级管理员'), value: '100' },
    ],
    [t],
  );
  const statusOptions = useMemo(
    () => [
      { label: t('全部状态'), value: '' },
      { label: t('已启用'), value: '1' },
      { label: t('已禁用'), value: '2' },
    ],
    [t],
  );
  // 布尔筛选使用明确标签，便于一眼区分。
  const hasInviterOptions = useMemo(
    () => [
      { label: t('邀请人：全部'), value: '' },
      { label: t('邀请人：有'), value: 'true' },
      { label: t('邀请人：无'), value: 'false' },
    ],
    [t],
  );
  const hasInviteesOptions = useMemo(
    () => [
      { label: t('下游邀请：全部'), value: '' },
      { label: t('下游邀请：有'), value: 'true' },
      { label: t('下游邀请：无'), value: 'false' },
    ],
    [t],
  );
  const hasActiveSubscriptionOptions = useMemo(
    () => [
      { label: t('套餐：全部'), value: '' },
      { label: t('套餐：有生效套餐'), value: 'true' },
      { label: t('套餐：无生效套餐'), value: 'false' },
    ],
    [t],
  );
  const hasSellableTokenOptions = useMemo(
    () => [
      { label: t('令牌情况：全部'), value: '' },
      { label: t('令牌情况：有'), value: 'true' },
      { label: t('令牌情况：无'), value: 'false' },
    ],
    [t],
  );
  // 两种排序可同时生效（例如 ID 降序 + 余额升序）。
  // ID 与余额排序使用独立文案，避免 "asc/desc" 歧义。
  const idSortDirectionOptions = useMemo(
    () => [
      { label: t('ID排序：默认'), value: '' },
      { label: t('ID排序：升序'), value: 'asc' },
      { label: t('ID排序：降序'), value: 'desc' },
    ],
    [t],
  );
  const balanceSortDirectionOptions = useMemo(
    () => [
      { label: t('余额排序：默认'), value: '' },
      { label: t('余额排序：升序'), value: 'asc' },
      { label: t('余额排序：降序'), value: 'desc' },
    ],
    [t],
  );

  // 统计已启用的高级筛选数量，用于徽标提示隐藏条件。
  const activeAdvancedCount = useMemo(() => {
    if (!advancedFilters) return 0;
    return Object.values(advancedFilters).filter(
      (value) => value !== '' && value !== null && value !== undefined,
    ).length;
  }, [advancedFilters]);

  const openAdvancedFilters = () => {
    // 打开时用当前生效值重建草稿筛选，保证取消不会影响在线筛选状态。
    setDraftAdvancedFilters({
      ...(defaultAdvancedFilters || {}),
      ...(advancedFilters || {}),
    });
    setAdvancedVisible(true);
  };

  const applyAdvanced = async () => {
    await applyAdvancedFilters?.(draftAdvancedFilters);
    setAdvancedVisible(false);
  };

  const resetDraftAdvanced = () => {
    setDraftAdvancedFilters(defaultAdvancedFilters || {});
  };

  const handleReset = async () => {
    if (formApiRef.current) {
      formApiRef.current.reset();
    }
    // 重置基础与高级筛选，避免重置后仍有后端隐形条件。
    await resetAdvancedFilters?.();
  };

  return (
    <>
      <Form
        initValues={formInitValues}
        getFormApi={(api) => {
          setFormApi(api);
          formApiRef.current = api;
        }}
        onSubmit={() => {
          searchUsers(1, pageSize);
        }}
        allowEmpty={true}
        autoComplete='off'
        layout='horizontal'
        trigger='change'
        stopValidateWithError={false}
        className='w-full md:w-auto order-1 md:order-2'
      >
        <div className='flex flex-col md:flex-row md:flex-wrap items-center gap-2 w-full md:w-auto'>
          <div className='relative w-full md:w-64'>
            <Form.Input
              field='searchKeyword'
              prefix={<IconSearch />}
              // placeholder 聚焦常用字段，减少视觉噪音。
              placeholder={t('ID/用户名')}
              showClear
              pure
              size='small'
            />
          </div>
          <div className='w-full md:w-48'>
            <Form.Select
              field='searchGroup'
              placeholder={t('选择分组')}
              optionList={groupOptions}
              onChange={() => {
                // Group 是高频项，保持即时搜索以减少一次点击。
                setTimeout(() => {
                  searchUsers(1, pageSize);
                }, 100);
              }}
              className='w-full'
              showClear
              pure
              size='small'
            />
          </div>
          <div className='flex gap-2 w-full md:w-auto'>
            <Button
              type='tertiary'
              htmlType='submit'
              loading={loading || searching}
              className='flex-1 md:flex-initial md:w-auto'
              size='small'
            >
              {t('查询')}
            </Button>
            <Button
              type='tertiary'
              onClick={openAdvancedFilters}
              className='flex-1 md:flex-initial md:w-auto'
              icon={<IconFilter />}
              size='small'
            >
              {activeAdvancedCount > 0
                ? `${t('高级筛选')}(${activeAdvancedCount})`
                : t('高级筛选')}
            </Button>
            <Button
              type='tertiary'
              onClick={handleReset}
              className='flex-1 md:flex-initial md:w-auto'
              size='small'
            >
              {t('重置')}
            </Button>
          </div>
        </div>
      </Form>

      <SideSheet
        visible={advancedVisible}
        onCancel={() => setAdvancedVisible(false)}
        title={t('高级筛选')}
        placement='right'
        width={460}
        bodyStyle={{ padding: 16 }}
        footer={
          <div className='flex justify-end'>
            <Space>
              <Button type='tertiary' onClick={resetDraftAdvanced}>
                {t('清空当前设置')}
              </Button>
              <Button type='primary' theme='solid' onClick={applyAdvanced}>
                {t('应用筛选')}
              </Button>
            </Space>
          </div>
        }
      >
        {/* 将高级筛选拆分为基础/额度/排序分区，减少表单拥挤。 */}
        <div className='grid grid-cols-1 gap-3'>
          <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
            <Text type='tertiary' size='small'>
              {t('基础筛选')}
            </Text>
            <div className='grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2'>
              <Select
                value={draftAdvancedFilters.searchRole}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchRole: value,
                  }))
                }
                optionList={roleOptions}
                placeholder={t('角色')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchStatus}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchStatus: value,
                  }))
                }
                optionList={statusOptions}
                placeholder={t('状态')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchHasInviter}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchHasInviter:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={hasInviterOptions}
                placeholder={t('是否有邀请人')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchHasInvitees}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchHasInvitees:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={hasInviteesOptions}
                placeholder={t('是否有被邀请人')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchHasActiveSubscription}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchHasActiveSubscription:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={hasActiveSubscriptionOptions}
                placeholder={t('套餐筛选')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchHasSellableToken}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchHasSellableToken:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={hasSellableTokenOptions}
                placeholder={t('令牌情况筛选')}
                showClear
              />
              <InputNumber
                value={
                  draftAdvancedFilters.searchInviterId === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchInviterId)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchInviterId:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={1}
                precision={0}
                placeholder={t('邀请人 ID')}
                style={{ width: '100%' }}
              />
              <InputNumber
                value={
                  draftAdvancedFilters.searchInviteeUserId === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchInviteeUserId)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchInviteeUserId:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={1}
                precision={0}
                placeholder={t('被邀请人 ID')}
                style={{ width: '100%' }}
              />
            </div>
          </div>

          <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
            <Text type='tertiary' size='small'>
              {t('额度筛选')}
            </Text>
            <div className='grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2'>
              <InputNumber
                value={
                  draftAdvancedFilters.searchBalanceMin === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchBalanceMin)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchBalanceMin:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={0}
                precision={0}
                placeholder={t('额度最小值')}
                style={{ width: '100%' }}
              />
              <InputNumber
                value={
                  draftAdvancedFilters.searchBalanceMax === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchBalanceMax)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchBalanceMax:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={0}
                precision={0}
                placeholder={t('额度最大值')}
                style={{ width: '100%' }}
              />
              <Divider margin='4px' className='sm:col-span-2' />
              <InputNumber
                value={
                  draftAdvancedFilters.searchUsedBalanceMin === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchUsedBalanceMin)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchUsedBalanceMin:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={0}
                precision={0}
                placeholder={t('已使用余额最小值')}
                style={{ width: '100%' }}
              />
              <InputNumber
                value={
                  draftAdvancedFilters.searchUsedBalanceMax === ''
                    ? undefined
                    : Number(draftAdvancedFilters.searchUsedBalanceMax)
                }
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchUsedBalanceMax:
                      value === null || value === undefined ? '' : String(value),
                  }))
                }
                min={0}
                precision={0}
                placeholder={t('已使用余额最大值')}
                style={{ width: '100%' }}
              />
            </div>
          </div>

          <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
            <Text type='tertiary' size='small'>
              {t('高级排序')}
            </Text>
            <div className='grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2'>
              <Select
                value={draftAdvancedFilters.searchIdSortOrder}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchIdSortOrder:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={idSortDirectionOptions}
                placeholder={t('ID排序')}
                showClear
              />
              <Select
                value={draftAdvancedFilters.searchBalanceSortOrder}
                onChange={(value) =>
                  setDraftAdvancedFilters((prev) => ({
                    ...prev,
                    searchBalanceSortOrder:
                      value === null || value === undefined ? '' : value,
                  }))
                }
                optionList={balanceSortDirectionOptions}
                placeholder={t('余额排序')}
                showClear
              />
            </div>
          </div>
        </div>
      </SideSheet>
    </>
  );
};

export default UsersFilters;
