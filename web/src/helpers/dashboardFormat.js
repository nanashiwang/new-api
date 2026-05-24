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

const baseColors = [
  '#1664FF',
  '#1AC6FF',
  '#FF8A00',
  '#3CC780',
  '#7442D4',
  '#FFC400',
  '#304D77',
  '#B48DEB',
  '#009488',
  '#FF7DDA',
];

const extendedColors = [
  '#1664FF',
  '#B2CFFF',
  '#1AC6FF',
  '#94EFFF',
  '#FF8A00',
  '#FFCE7A',
  '#3CC780',
  '#B9EDCD',
  '#7442D4',
  '#DDC5FA',
  '#FFC400',
  '#FAE878',
  '#304D77',
  '#8B959E',
  '#B48DEB',
  '#EFE3FF',
  '#009488',
  '#59BAA8',
  '#FF7DDA',
  '#FFCFEE',
];

export const modelColorMap = {
  'dall-e': 'rgb(147,112,219)',
  'dall-e-3': 'rgb(153,50,204)',
  'gpt-3.5-turbo': 'rgb(184,227,167)',
  'gpt-3.5-turbo-0613': 'rgb(60,179,113)',
  'gpt-3.5-turbo-1106': 'rgb(32,178,170)',
  'gpt-3.5-turbo-16k': 'rgb(149,252,206)',
  'gpt-3.5-turbo-16k-0613': 'rgb(119,255,214)',
  'gpt-3.5-turbo-instruct': 'rgb(175,238,238)',
  'gpt-4': 'rgb(135,206,235)',
  'gpt-4-0613': 'rgb(100,149,237)',
  'gpt-4-1106-preview': 'rgb(30,144,255)',
  'gpt-4-0125-preview': 'rgb(2,177,236)',
  'gpt-4-turbo-preview': 'rgb(2,177,255)',
  'gpt-4-32k': 'rgb(104,111,238)',
  'gpt-4-32k-0613': 'rgb(61,71,139)',
  'gpt-4-all': 'rgb(65,105,225)',
  'gpt-4-gizmo-*': 'rgb(0,0,255)',
  'gpt-4-vision-preview': 'rgb(25,25,112)',
  'text-ada-001': 'rgb(255,192,203)',
  'text-babbage-001': 'rgb(255,160,122)',
  'text-curie-001': 'rgb(219,112,147)',
  'text-davinci-003': 'rgb(219,112,147)',
  'text-davinci-edit-001': 'rgb(255,105,180)',
  'text-embedding-ada-002': 'rgb(255,182,193)',
  'text-embedding-v1': 'rgb(255,174,185)',
  'text-moderation-latest': 'rgb(255,130,171)',
  'text-moderation-stable': 'rgb(255,160,122)',
  'tts-1': 'rgb(255,140,0)',
  'tts-1-1106': 'rgb(255,165,0)',
  'tts-1-hd': 'rgb(255,215,0)',
  'tts-1-hd-1106': 'rgb(255,223,0)',
  'whisper-1': 'rgb(245,245,220)',
  'claude-3-opus-20240229': 'rgb(255,132,31)',
  'claude-3-sonnet-20240229': 'rgb(253,135,93)',
  'claude-3-haiku-20240307': 'rgb(255,175,146)',
};

export function modelToColor(modelName = '') {
  if (modelColorMap[modelName]) {
    return modelColorMap[modelName];
  }

  let hash = 0;
  for (let i = 0; i < modelName.length; i++) {
    hash = (hash << 5) - hash + modelName.charCodeAt(i);
    hash = hash & hash;
  }

  const colorPalette = modelName.length > 10 ? extendedColors : baseColors;
  return colorPalette[Math.abs(hash) % colorPalette.length];
}

const avatarColors = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];

export function stringToColor(str = '') {
  let sum = 0;
  for (let i = 0; i < str.length; i++) {
    sum += str.charCodeAt(i);
  }
  return avatarColors[sum % avatarColors.length];
}

export function renderNumber(num) {
  if (num >= 1000000000) {
    return (num / 1000000000).toFixed(1) + 'B';
  } else if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  } else if (num >= 10000) {
    return (num / 1000).toFixed(1) + 'k';
  }
  return num;
}

export function getQuotaWithUnit(quota, digits = 6) {
  const quotaPerUnit = parseFloat(localStorage.getItem('quota_per_unit'));
  return (quota / quotaPerUnit).toFixed(digits);
}

export function renderQuota(quota, digits = 2) {
  const quotaPerUnit = parseFloat(localStorage.getItem('quota_per_unit'));
  const quotaDisplayType = localStorage.getItem('quota_display_type') || 'USD';
  if (quotaDisplayType === 'TOKENS') {
    return renderNumber(quota);
  }

  const resultUSD = quota / quotaPerUnit;
  let symbol = '$';
  let value = resultUSD;

  if (quotaDisplayType === 'CNY') {
    const statusStr = localStorage.getItem('status');
    let usdRate = 1;
    try {
      if (statusStr) {
        const status = JSON.parse(statusStr);
        usdRate = status?.usd_exchange_rate || 1;
      }
    } catch (e) {}
    value = resultUSD * usdRate;
    symbol = '\u00a5';
  } else if (quotaDisplayType === 'CUSTOM') {
    const statusStr = localStorage.getItem('status');
    let symbolCustom = '\u00a4';
    let rate = 1;
    try {
      if (statusStr) {
        const status = JSON.parse(statusStr);
        symbolCustom = status?.custom_currency_symbol || symbolCustom;
        rate = status?.custom_currency_exchange_rate || rate;
      }
    } catch (e) {}
    value = resultUSD * rate;
    symbol = symbolCustom;
  }

  const fixedResult = value.toFixed(digits);
  if (parseFloat(fixedResult) === 0 && quota > 0 && value > 0) {
    const minValue = Math.pow(10, -digits);
    return symbol + minValue.toFixed(digits);
  }
  return symbol + fixedResult;
}
