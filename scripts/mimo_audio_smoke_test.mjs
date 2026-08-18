#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';

const baseURL = (process.env.NAN_BASE_URL || 'https://cn.meta-api.vip').replace(/\/+$/, '');
const apiKey = (process.env.NAN_API_KEY || '').trim();
const outputDir = path.resolve(process.env.MIMO_OUTPUT_DIR || './mimo-audio-output');
const ttsVoice = process.env.MIMO_TTS_VOICE || '冰糖';
const endpoint = `${baseURL}/v1/chat/completions`;

if (!apiKey) {
  console.error('缺少 NAN_API_KEY，请先设置平台 API Key。');
  process.exit(1);
}

fs.mkdirSync(outputDir, { recursive: true });

function sanitize(value) {
  if (!value || typeof value !== 'object') return value;
  if (Array.isArray(value)) return value.map(sanitize);

  const output = {};
  for (const [key, item] of Object.entries(value)) {
    if (key === 'data' && typeof item === 'string' && item.length > 200) {
      output[key] = `<base64:${item.length} chars>`;
    } else if (key === 'voice' && typeof item === 'string' && item.startsWith('data:')) {
      output[key] = `<audio-data-uri:${item.length} chars>`;
    } else {
      output[key] = sanitize(item);
    }
  }
  return output;
}

async function callModel(name, payload) {
  const startedAt = Date.now();
  let response;
  let rawBody;

  try {
    response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
      signal: AbortSignal.timeout(180_000),
    });
    rawBody = await response.text();
  } catch (error) {
    throw new Error(`${name} 网络请求失败：${error}`);
  }

  let body;
  try {
    body = JSON.parse(rawBody);
  } catch {
    body = { raw: rawBody.slice(0, 2_000) };
  }

  const result = {
    name,
    ok: response.ok,
    status: response.status,
    elapsed_ms: Date.now() - startedAt,
    error: body?.error || null,
    body: sanitize(body),
  };

  fs.writeFileSync(
    path.join(outputDir, `${name}.json`),
    `${JSON.stringify(result, null, 2)}\n`,
  );

  console.log(
    `${result.ok ? '✅' : '❌'} ${name}: HTTP ${result.status}, ${(result.elapsed_ms / 1000).toFixed(2)}s`,
  );

  if (!response.ok) {
    throw new Error(`${name} 调用失败：${JSON.stringify(body?.error || body)}`);
  }

  return body;
}

function saveAudio(body, filename) {
  const data = body?.choices?.[0]?.message?.audio?.data;
  if (typeof data !== 'string' || data.length === 0) {
    throw new Error(`${filename} 未找到 choices[0].message.audio.data`);
  }

  const base64 = data.includes(',') ? data.slice(data.indexOf(',') + 1) : data;
  const bytes = Buffer.from(base64, 'base64');
  const outputPath = path.join(outputDir, filename);
  fs.writeFileSync(outputPath, bytes);
  console.log(`   音频已保存：${outputPath} (${bytes.length} bytes)`);
  return { base64, bytes, outputPath };
}

async function main() {
  console.log(`请求地址：${endpoint}`);
  console.log(`输出目录：${outputDir}`);

  const ttsBody = await callModel('01-tts', {
    model: 'mimo-v2.5-tts',
    messages: [
      {
        role: 'user',
        content: '请使用自然、清晰、温和的中文女声，语速适中。',
      },
      {
        role: 'assistant',
        content:
          '你好，这是小米 MiMo 语音合成模型的本地接口测试。我们正在验证语音生成、音色克隆和语音识别功能是否能够正常工作。',
      },
    ],
    audio: {
      format: 'wav',
      voice: ttsVoice,
    },
    stream: false,
  });
  const referenceAudio = saveAudio(ttsBody, '01-tts.wav');

  const voiceDesignBody = await callModel('02-voicedesign', {
    model: 'mimo-v2.5-tts-voicedesign',
    messages: [
      {
        role: 'user',
        content:
          '二十五岁左右的中国女性声音，温柔自然、清晰有亲和力，语速适中，录音棚音质。',
      },
      {
        role: 'assistant',
        content: '你好，这是一段根据文字描述设计出来的全新音色。',
      },
    ],
    audio: {
      format: 'wav',
      optimize_text_preview: true,
    },
    stream: false,
  });
  saveAudio(voiceDesignBody, '02-voicedesign.wav');

  const voiceCloneBody = await callModel('03-voiceclone', {
    model: 'mimo-v2.5-tts-voiceclone',
    messages: [
      {
        role: 'user',
        content: '保持参考声音的自然语气和正常语速。',
      },
      {
        role: 'assistant',
        content: '你好，这是使用参考音频完成的音色克隆测试。',
      },
    ],
    audio: {
      format: 'wav',
      voice: `data:audio/wav;base64,${referenceAudio.base64}`,
    },
    stream: false,
  });
  saveAudio(voiceCloneBody, '03-voiceclone.wav');

  const asrBody = await callModel('04-asr', {
    model: 'mimo-v2.5-asr',
    messages: [
      {
        role: 'user',
        content: [
          {
            type: 'input_audio',
            input_audio: {
              data: `data:audio/wav;base64,${referenceAudio.base64}`,
            },
          },
        ],
      },
    ],
    stream: false,
  });

  const transcript = asrBody?.choices?.[0]?.message?.content;
  console.log(`   ASR 识别结果：${transcript || '<空>'}`);
  console.log('全部测试完成。');
}

main().catch((error) => {
  console.error(error.message || error);
  process.exitCode = 1;
});
