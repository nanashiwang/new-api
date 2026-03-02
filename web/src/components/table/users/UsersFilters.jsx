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

  // 这里统一定义高级筛选下拉选项，避免在 JSX 中散落硬编码，后续迭代时可直接扩展。
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
  // 两个“布尔筛选”使用不同文案，避免都显示“全部/是/否”时用户无法区分当前筛选含义。
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
  // 两个排序项可同时生效：例如 ID 降序 + 余额升序。
  // 这里分别给 ID/余额提供独立文案，避免都显示“升序/降序”导致语义不清。
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

  // 统计激活中的高级筛选数量，用于按钮上的数字提示，帮助管理员识别当前是否存在隐藏过滤条件。
  const activeAdvancedCount = useMemo(() => {
    if (!advancedFilters) return 0;
    return Object.values(advancedFilters).filter(
      (value) => value !== '' && value !== null && value !== undefined,
    ).length;
  }, [advancedFilters]);

  const openAdvancedFilters = () => {
    // 每次打开抽屉都以“当前生效值”为准初始化草稿，保证取消操作不会污染已生效条件。
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
    // 这里重置“基础搜索 + 高级筛选”两类条件，避免出现只清空了可见条件但后台仍带高级过滤的问题。
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
              // 输入提示按你的要求收敛为最常用字段，避免视觉干扰。
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
                // 分组是高频条件，保留即时触发查询，减少一次额外点击。
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
        width={420}
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
        <div className='grid grid-cols-1 gap-3'>
          <Text type='tertiary' size='small'>
            {t('基础筛选')}
          </Text>
          <Select
            value={draftAdvancedFilters.searchRole}
            onChange={(value) =>
              setDraftAdvancedFilters((prev) => ({ ...prev, searchRole: value }))
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
          <Divider margin='4px' />
          <Text type='tertiary' size='small'>
            {t('额度筛选')}
          </Text>
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
          <Divider margin='4px' />
          <Text type='tertiary' size='small'>
            {t('已使用余额筛选')}
          </Text>
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
          <Divider margin='4px' />
          <Text type='tertiary' size='small'>
            {t('高级排序')}
          </Text>
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
      </SideSheet>
    </>
  );
};

export default UsersFilters;
