/**
 * bucketGuides.ts — 电商套图素材槽位的拍摄/选图教学内容。
 *
 * 设计原则：
 *  - 零额外图片素材：tip 卡用 emoji 表达概念，style 渐变意境卡用 Tailwind 渐变 class
 *  - 内容静态化，便于未来 i18n（如果项目接入翻译框架）
 *  - 结构化数据，Phase 2 准备真实样图后只需给 StyleSample 加 sampleUrl 字段
 */
import type { ProductAssetKind } from '../types'

export interface BucketGuideTip {
  /** 概念图标，使用 emoji 字符串避免新增 SVG 资源 */
  emoji: string
  title: string
  description: string
}

export interface StyleSample {
  /** 对应 STYLE_PRESETS 的 id，便于 Phase 2 与"应用此风格示范"联动 */
  presetId: string
  name: string
  description: string
  /** Tailwind class，用作意境卡背景，传达风格氛围；Phase 2 可改成真实 sampleUrl */
  gradient: string
}

export interface BucketGuide {
  kind: ProductAssetKind
  title: string
  intro: string
  /** 非 style 槽位使用 */
  tips?: BucketGuideTip[]
  /** 仅 style 槽位使用 */
  styleSamples?: StyleSample[]
}

export const BUCKET_GUIDES: Record<ProductAssetKind, BucketGuide> = {
  main: {
    kind: 'main',
    title: '主商品图建议',
    intro: '主商品图是所有套图的主体参考，gpt-image-2 会以它为基准保持商品外观一致。',
    tips: [
      { emoji: '🎯', title: '正面清晰', description: '正面平视拍摄，避免畸变；商品占画面 60-70%。' },
      { emoji: '💡', title: '光线均匀', description: '柔和自然光或专业柔光箱，避免硬阴影遮挡细节。' },
      { emoji: '🧹', title: '背景干净', description: '纯白/浅灰背景最佳，避免杂物干扰主体识别。' },
      { emoji: '🚫', title: '无水印贴纸', description: '去除价格标签、防伪贴、平台水印，否则会被沿用到所有套图。' },
    ],
  },
  angle: {
    kind: 'angle',
    title: '多角度图建议',
    intro: '补充主图未覆盖的视角，让模型理解商品的立体结构与各方位细节。',
    tips: [
      { emoji: '📷', title: '正面平视', description: '与主图视角一致，可上传更高分辨率版本。' },
      { emoji: '📐', title: '侧 45°', description: '体现立体感、轮廓线条与厚度。' },
      { emoji: '🔍', title: '细节特写', description: '工艺、材质、关键卖点（如表盘、面料纹理、镜头）。' },
      { emoji: '🔄', title: '背面/侧面', description: '包装信息、接口、按键、商标位置等。' },
      { emoji: '📦', title: '包装合影', description: '商品 + 原盒/说明书/赠品，强化"全套"真实感。' },
    ],
  },
  sku: {
    kind: 'sku',
    title: '规格 / SKU 图建议',
    intro: '展示同款的颜色、容量、型号差异，让模型在生成 SKU 套图时保持一致性。',
    tips: [
      { emoji: '🎨', title: '多色平铺', description: '同款不同色一字排开，等距、等角度、相同光线。' },
      { emoji: '📊', title: '规格对比', description: '大/中/小或不同容量并排，体现尺寸关系。' },
      { emoji: '👥', title: '全 SKU 合影', description: '系列全员同框，便于规划"全规格图"。' },
      { emoji: '🎁', title: '套装组合', description: '礼盒、配件、赠品组合，适合促销场景。' },
    ],
  },
  brand: {
    kind: 'brand',
    title: '品牌素材建议',
    intro: '提供品牌识别元素，模型在生成宣传图时会保持品牌视觉一致性。',
    tips: [
      { emoji: '🏷️', title: 'Logo', description: '透明背景 PNG/SVG，主版+反白版+小尺寸版。' },
      { emoji: '🎨', title: '品牌色卡', description: '主色 + 辅色 + 强调色，标注 HEX 或 Pantone 值。' },
      { emoji: '✏️', title: '字体规范', description: '标题字体与正文字体样例图，含中英文。' },
      { emoji: '📦', title: '包装实拍', description: '标准包装的真实照片，含 Logo、色彩、版式细节。' },
    ],
  },
  style: {
    kind: 'style',
    title: '风格参考图说明',
    intro: '上传你想模仿的视觉风格参考图（光线、构图、质感），模型会借鉴这些特征生成新图。下方为 7 种内置风格的意境示意，对应右侧「风格模板」下拉选项。',
    styleSamples: [
      {
        presetId: 'minimal-white',
        name: '极简白底电商风',
        description: '纯白/浅灰背景，商品居中，干净阴影，适合通用主图。',
        gradient: 'bg-gradient-to-br from-gray-50 to-gray-200',
      },
      {
        presetId: 'tech-premium',
        name: '高级科技感',
        description: '冷色调、金属质感、科技光效，适合数码家电。',
        gradient: 'bg-gradient-to-br from-slate-700 via-slate-900 to-blue-950',
      },
      {
        presetId: 'xiaohongshu-life',
        name: '小红书生活方式',
        description: '柔光、真实生活场景、低饱和、种草感。',
        gradient: 'bg-gradient-to-br from-rose-100 via-amber-100 to-orange-200',
      },
      {
        presetId: 'livestream-hit',
        name: '直播间爆款风',
        description: '强卖点、强对比、促销氛围，适合直播与活动。',
        gradient: 'bg-gradient-to-br from-red-500 via-orange-400 to-yellow-300',
      },
      {
        presetId: 'outdoor-scene',
        name: '户外场景风',
        description: '自然光、户外背景、动感构图，适合运动露营车载。',
        gradient: 'bg-gradient-to-br from-emerald-200 via-sky-300 to-blue-400',
      },
      {
        presetId: 'luxury-texture',
        name: '奢华质感风',
        description: '深色背景、精致高光、礼盒/首饰/香水质感。',
        gradient: 'bg-gradient-to-br from-zinc-900 via-stone-800 to-amber-700',
      },
      {
        presetId: 'japanese-fresh',
        name: '日系清新风',
        description: '浅色、留白、清爽、低饱和，适合家居文具小物。',
        gradient: 'bg-gradient-to-br from-stone-50 via-emerald-50 to-sky-100',
      },
    ],
  },
}

export function getBucketGuide(kind: ProductAssetKind): BucketGuide {
  return BUCKET_GUIDES[kind]
}
