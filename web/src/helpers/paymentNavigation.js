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

const isPaymentUrl = (value) => {
  if (typeof value !== 'string' || !value.trim()) return false;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' || url.protocol === 'http:';
  } catch {
    return false;
  }
};

// Payment URLs arrive after an async order request. Same-tab navigation does
// not depend on the transient user activation needed to open a new window.
export const redirectToPayment = (url) => {
  if (!isPaymentUrl(url)) return false;
  window.location.assign(url);
  return true;
};

export const submitPaymentForm = (url, params) => {
  if (
    !isPaymentUrl(url) ||
    !params ||
    typeof params !== 'object' ||
    Array.isArray(params) ||
    Object.keys(params).length === 0
  ) {
    return false;
  }
  const form = document.createElement('form');
  form.action = url;
  form.method = 'POST';
  form.target = '_self';
  form.hidden = true;
  try {
    for (const [key, value] of Object.entries(params)) {
      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = key;
      input.value = value;
      form.appendChild(input);
    }
    document.body.appendChild(form);
    // A provider field named "submit" must not shadow the native method.
    HTMLFormElement.prototype.submit.call(form);
    return true;
  } finally {
    form.remove();
  }
};
