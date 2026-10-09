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
import { Banner } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { getEpayCryptoMethod } from '../../helpers/epayMethods';

export default function CryptoPaymentNotice({ method }) {
  const { t } = useTranslation();
  const crypto = getEpayCryptoMethod(method);
  if (!crypto) return null;
  return (
    <Banner
      type='info'
      closeIcon={null}
      description={t(
        '订单以人民币计价。请使用 {{network}} 网络支付 {{coin}}，并核对收银台的代币合约、地址和精确数量；到账确认后自动更新订单，请勿跨链或重复转账。',
        { network: crypto.network, coin: crypto.coin },
      )}
    />
  );
}
