import assert from 'node:assert/strict';
import fs from 'node:fs';
import { test } from 'bun:test';

const source = fs.readFileSync(new URL('./PaymentRiskCaseDetailModal.jsx', import.meta.url), 'utf8');

test('PaymentRiskCaseDetailModal imports TextArea from the supported Semi UI export', () => {
  assert.match(
    source,
    /import\s*{[^}]*\bTextArea\b[^}]*}\s*from\s*'@douyinfe\/semi-ui';/,
    'PaymentRiskCaseDetailModal should import TextArea directly from @douyinfe/semi-ui',
  );

  assert.doesNotMatch(
    source,
    /const\s*{\s*TextArea\s*}\s*=\s*Input\s*;/,
    'PaymentRiskCaseDetailModal should not read TextArea from Input; this Semi UI version does not expose Input.TextArea',
  );
});
