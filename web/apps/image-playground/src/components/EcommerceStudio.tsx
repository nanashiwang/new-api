import { useEffect, useMemo, useRef, useState, type ButtonHTMLAttributes, type ReactNode } from 'react'
import type { EcommerceBrief, ProductAsset, ProductAssetKind, SuitePlanGroup, SuitePlanItem, TaskRecord } from '../types'
import {
  createInputImageFromFile,
  ensureImageCached,
  ensureImageThumbnailCached,
  subscribeImageThumbnail,
  useStore,
} from '../store'
import {
  ECOMMERCE_GROUPS,
  ECOMMERCE_PLATFORMS,
  PRODUCT_ASSET_BUCKETS,
  SIZE_PRESETS,
  STYLE_PRESETS,
  SUITE_TEMPLATES,
  getSizePreset,
  getStylePreset,
  getSuiteTemplate,
} from '../lib/ecommerce'
import { CopyIcon, DownloadIcon, EditIcon, PhotoIcon, PlusIcon, RefreshIcon, TrashIcon } from './icons'

function parseTags(value: string) {
  return value.split(/[，,、\n]/).map((item) => item.trim()).filter(Boolean)
}

function statusLabel(item: SuitePlanItem) {
  if (item.status === 'generating') return '生成中'
  if (item.status === 'done') return '已完成'
  if (item.status === 'partial_success') return '部分成功'
  if (item.status === 'error') return '失败'
  return '草稿'
}

function statusClass(item: SuitePlanItem) {
  if (item.status === 'done') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  if (item.status === 'generating') return 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300'
  if (item.status === 'partial_success') return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  if (item.status === 'error') return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-white/[0.06] dark:text-gray-300'
}

function ImageThumb({ imageId, className = '' }: { imageId: string; className?: string }) {
  const [src, setSrc] = useState('')

  useEffect(() => {
    let mounted = true
    ensureImageThumbnailCached(imageId).then((thumb) => {
      if (mounted && thumb?.dataUrl) setSrc(thumb.dataUrl)
    })
    const unsubscribe = subscribeImageThumbnail(imageId, (thumb) => setSrc(thumb.dataUrl))
    return () => {
      mounted = false
      unsubscribe()
    }
  }, [imageId])

  if (!src) {
    return (
      <div className={`grid place-items-center bg-gray-100 dark:bg-white/[0.05] text-gray-400 ${className}`}>
        <PhotoIcon className="w-6 h-6" />
      </div>
    )
  }

  return <img src={src} alt="" className={`object-cover ${className}`} draggable={false} />
}

function FieldLabel({ children }: { children: ReactNode }) {
  return <label className="text-xs font-semibold text-gray-500 dark:text-gray-400">{children}</label>
}

function TextInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full rounded-xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-gray-900/10 dark:focus:ring-white/10 ${props.className ?? ''}`}
    />
  )
}

function TextArea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`w-full rounded-xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-gray-900/10 dark:focus:ring-white/10 ${props.className ?? ''}`}
    />
  )
}

function SelectInput(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`w-full rounded-xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-gray-900/10 dark:focus:ring-white/10 ${props.className ?? ''}`}
    />
  )
}

function SwitchRow({ label, checked, onChange, hint }: { label: string; checked: boolean; onChange: (value: boolean) => void; hint?: string }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-xl border border-gray-200 dark:border-white/[0.08] bg-white/70 dark:bg-white/[0.03] px-3 py-2">
      <span>
        <span className="block text-sm font-medium text-gray-800 dark:text-gray-100">{label}</span>
        {hint && <span className="block text-xs text-gray-500 mt-0.5">{hint}</span>}
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="h-4 w-4 accent-gray-900 dark:accent-white"
      />
    </label>
  )
}

function BucketCard({
  kind,
  assets,
  onUpload,
  onRemove,
}: {
  kind: ProductAssetKind
  assets: ProductAsset[]
  onUpload: (kind: ProductAssetKind, files: FileList | null) => void
  onRemove: (assetId: string) => void
}) {
  const bucket = PRODUCT_ASSET_BUCKETS.find((item) => item.kind === kind)!
  const inputRef = useRef<HTMLInputElement>(null)
  const reachedLimit = assets.length >= bucket.max

  return (
    <section className="rounded-2xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 p-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-gray-900 dark:text-white">
            {bucket.label}{bucket.required && <span className="text-red-500 ml-1">*</span>}
          </div>
          <div className="text-xs text-gray-500 mt-1">{bucket.hint}</div>
        </div>
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={reachedLimit}
          className="shrink-0 inline-flex items-center gap-1 rounded-lg bg-gray-900 text-white dark:bg-white dark:text-gray-950 px-2 py-1 text-xs font-medium disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <PlusIcon className="w-3.5 h-3.5" />
          上传
        </button>
      </div>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        multiple={kind !== 'main'}
        className="hidden"
        onChange={(event) => {
          onUpload(kind, event.currentTarget.files)
          event.currentTarget.value = ''
        }}
      />
      <div className="mt-3 grid grid-cols-3 gap-2">
        {assets.map((asset) => (
          <div key={asset.id} className="relative group overflow-hidden rounded-xl border border-gray-200 dark:border-white/[0.08]">
            <ImageThumb imageId={asset.imageId} className="aspect-square w-full" />
            <button
              type="button"
              onClick={() => onRemove(asset.id)}
              className="absolute right-1 top-1 rounded-md bg-black/55 p-1 text-white opacity-0 transition-opacity group-hover:opacity-100"
              aria-label="删除素材"
            >
              <TrashIcon className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
        {assets.length === 0 && (
          <div className="col-span-3 rounded-xl border border-dashed border-gray-200 dark:border-white/[0.08] py-5 text-center text-xs text-gray-400">
            未上传
          </div>
        )}
      </div>
      <div className="mt-2 text-[11px] text-gray-400">{assets.length}/{bucket.max}</div>
    </section>
  )
}

function PlanCard({
  item,
  tasks,
  selected,
  onSelect,
  onPatch,
  onRetry,
  onCopyPrompt,
  onSetStyle,
  onDownload,
  onEdit,
  onOpenTask,
}: {
  item: SuitePlanItem
  tasks: TaskRecord[]
  selected: boolean
  onSelect: () => void
  onPatch: (patch: Partial<SuitePlanItem>) => void
  onRetry: () => void
  onCopyPrompt: () => void
  onSetStyle: (imageId: string) => void
  onDownload: (imageId: string) => void
  onEdit: (imageId: string) => void
  onOpenTask: (taskId: string) => void
}) {
  const outputImageIds = tasks.flatMap((task) => task.outputImages)

  return (
    <article className={`rounded-2xl border bg-white dark:bg-gray-950 p-4 transition-colors ${selected ? 'border-gray-900 dark:border-white' : 'border-gray-200 dark:border-white/[0.08]'}`}>
      <div className="flex items-start justify-between gap-3">
        <button type="button" onClick={onSelect} className="min-w-0 text-left">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold text-gray-900 dark:text-white">{item.title}</span>
            <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${statusClass(item)}`}>{statusLabel(item)}</span>
          </div>
          <p className="mt-1 text-xs text-gray-500 line-clamp-2">{item.purpose}</p>
        </button>
        <label className="inline-flex items-center gap-1 text-xs text-gray-500">
          <input type="checkbox" checked={item.enabled} onChange={(event) => onPatch({ enabled: event.target.checked })} />
          启用
        </label>
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-gray-500">
        <span className="rounded-full bg-gray-100 dark:bg-white/[0.06] px-2 py-1">比例 {item.ratio}</span>
        <label className="inline-flex items-center gap-1 rounded-full bg-gray-100 dark:bg-white/[0.06] px-2 py-1">
          数量
          <input
            type="number"
            min={1}
            max={6}
            value={item.count}
            onChange={(event) => onPatch({ count: Math.max(1, Math.min(6, Number(event.target.value) || 1)) })}
            className="w-10 bg-transparent text-center outline-none"
          />
        </label>
      </div>
      {item.error && <div className="mt-3 rounded-xl bg-red-50 dark:bg-red-500/10 px-3 py-2 text-xs text-red-600 dark:text-red-300">{item.error}</div>}
      {outputImageIds.length > 0 ? (
        <div className="mt-3 grid grid-cols-2 sm:grid-cols-3 gap-2">
          {outputImageIds.map((imageId) => (
            <div key={imageId} className="group relative overflow-hidden rounded-xl border border-gray-200 dark:border-white/[0.08]">
              <button type="button" onClick={() => onOpenTask(tasks.find((task) => task.outputImages.includes(imageId))?.id || '')} className="block w-full">
                <ImageThumb imageId={imageId} className="aspect-square w-full" />
              </button>
              <div className="absolute inset-x-1 bottom-1 flex justify-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                <IconButton label="风格参考" onClick={() => onSetStyle(imageId)}><PhotoIcon className="w-3.5 h-3.5" /></IconButton>
                <IconButton label="下载" onClick={() => onDownload(imageId)}><DownloadIcon className="w-3.5 h-3.5" /></IconButton>
                <IconButton label="遮罩编辑" onClick={() => onEdit(imageId)}><EditIcon className="w-3.5 h-3.5" /></IconButton>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="mt-3 rounded-xl border border-dashed border-gray-200 dark:border-white/[0.08] px-3 py-5 text-center text-xs text-gray-400">
          生成后图片会显示在这里
        </div>
      )}
      <div className="mt-3 flex flex-wrap gap-2">
        <SmallButton onClick={onRetry} disabled={!item.enabled || item.status === 'generating'} icon={<RefreshIcon className="w-3.5 h-3.5" />}>同类型重生成</SmallButton>
        <SmallButton onClick={onCopyPrompt} icon={<CopyIcon className="w-3.5 h-3.5" />}>复制提示词</SmallButton>
        <SmallButton disabled title="即将支持">设为主图</SmallButton>
        <SmallButton disabled title="即将支持">替换文案</SmallButton>
      </div>
      {selected && (
        <details className="mt-3 rounded-xl bg-gray-50 dark:bg-white/[0.04] p-3 text-xs text-gray-600 dark:text-gray-300">
          <summary className="cursor-pointer font-medium">查看最终 prompt 草案</summary>
          <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">{item.prompt || item.finalPromptDraft}</pre>
        </details>
      )}
    </article>
  )
}

function IconButton({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      title={label}
      onClick={(event) => {
        event.stopPropagation()
        onClick()
      }}
      className="rounded-lg bg-black/65 p-1.5 text-white backdrop-blur hover:bg-black/80"
    >
      {children}
    </button>
  )
}

function SmallButton({ children, icon, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { icon?: ReactNode }) {
  return (
    <button
      type="button"
      {...props}
      className={`inline-flex items-center gap-1 rounded-lg border border-gray-200 dark:border-white/[0.08] px-2.5 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-white/[0.04] disabled:cursor-not-allowed disabled:opacity-40 ${props.className ?? ''}`}
    >
      {icon}
      {children}
    </button>
  )
}

export default function EcommerceStudio() {
  const suites = useStore((s) => s.ecommerceSuites)
  const activeSuiteId = useStore((s) => s.activeEcommerceSuiteId)
  const selectedPlanItemId = useStore((s) => s.selectedEcommercePlanItemId)
  const planStatus = useStore((s) => s.ecommercePlanDraftStatus)
  const generationQueue = useStore((s) => s.generationQueue)
  const tasks = useStore((s) => s.tasks)
  const createSuite = useStore((s) => s.createEcommerceSuite)
  const updateBrief = useStore((s) => s.updateEcommerceBrief)
  const addAsset = useStore((s) => s.addEcommerceAsset)
  const removeAsset = useStore((s) => s.removeEcommerceAsset)
  const generatePlan = useStore((s) => s.generateEcommercePlan)
  const generateSuite = useStore((s) => s.generateEcommerceSuite)
  const retryPlanItem = useStore((s) => s.retryEcommercePlanItem)
  const updatePlanItem = useStore((s) => s.updateEcommercePlanItem)
  const setSelectedPlanItemId = useStore((s) => s.setSelectedEcommercePlanItemId)
  const addStyleReference = useStore((s) => s.addEcommerceStyleReference)
  const setMaskEditorImageId = useStore((s) => s.setMaskEditorImageId)
  const setDetailTaskId = useStore((s) => s.setDetailTaskId)
  const showToast = useStore((s) => s.showToast)
  const capabilities = useStore((s) => s.ecommerceCapabilities)
  const suite = suites.find((item) => item.id === activeSuiteId)

  useEffect(() => {
    if (!suite) createSuite()
  }, [suite, createSuite])

  const tasksByItem = useMemo(() => {
    const map = new Map<string, TaskRecord[]>()
    for (const task of tasks) {
      if (task.sourceMode !== 'ecommerce' || !task.suiteItemId || task.suiteId !== suite?.id) continue
      const list = map.get(task.suiteItemId) ?? []
      list.push(task)
      map.set(task.suiteItemId, list)
    }
    return map
  }, [tasks, suite?.id])

  if (!suite) {
    return <main className="safe-area-x max-w-7xl mx-auto py-10 text-sm text-gray-500">正在创建电商套图工程...</main>
  }

  const brief = suite.brief
  const style = getStylePreset(brief.stylePresetId)
  const template = getSuiteTemplate(brief.suiteTemplateId)
  const sizePreset = getSizePreset(brief.sizePreset)
  const isGenerating = generationQueue.length > 0

  const uploadAssets = async (kind: ProductAssetKind, files: FileList | null) => {
    if (!files?.length) return
    for (const file of Array.from(files)) {
      const image = await createInputImageFromFile(file)
      if (image) addAsset(kind, image.id, file.name)
    }
  }

  const copyPrompt = async (item: SuitePlanItem) => {
    await navigator.clipboard.writeText(item.prompt || item.finalPromptDraft)
    showToast('已复制提示词', 'success')
  }

  const downloadImage = async (imageId: string) => {
    const dataUrl = await ensureImageCached(imageId)
    if (!dataUrl) {
      showToast('图片已丢失', 'error')
      return
    }
    const a = document.createElement('a')
    a.href = dataUrl
    a.download = `ecommerce-${imageId.slice(0, 8)}.png`
    a.click()
  }

  const setStyle = (imageId: string) => {
    addStyleReference(imageId, '生成结果')
    showToast('已加入风格参考图', 'success')
  }

  return (
    <main className="safe-area-x max-w-[1600px] mx-auto pb-10">
      <div className="mb-4 rounded-3xl border border-gray-200 dark:border-white/[0.08] bg-gradient-to-r from-amber-50 via-white to-sky-50 dark:from-amber-500/10 dark:via-gray-950 dark:to-sky-500/10 p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-gray-500">Ecommerce Suite Studio</p>
            <h2 className="mt-1 text-2xl font-black tracking-tight text-gray-950 dark:text-white">电商套图工作台</h2>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">商品素材 → 套图方案 → 批量生成 → 回流画廊</p>
          </div>
          <div className="flex flex-wrap gap-2 text-xs text-gray-600 dark:text-gray-300">
            <span className="rounded-full bg-white/80 dark:bg-white/[0.06] px-3 py-1.5">规划模型 {capabilities.defaultPlanModel}</span>
            <span className="rounded-full bg-white/80 dark:bg-white/[0.06] px-3 py-1.5">出图模型 {capabilities.defaultImageModel}</span>
            <span className="rounded-full bg-white/80 dark:bg-white/[0.06] px-3 py-1.5">并发 2</span>
          </div>
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-[300px_minmax(0,1fr)_340px]">
        <aside className="space-y-3">
          {PRODUCT_ASSET_BUCKETS.map((bucket) => (
            <BucketCard
              key={bucket.kind}
              kind={bucket.kind}
              assets={suite.assets.filter((asset) => asset.kind === bucket.kind)}
              onUpload={uploadAssets}
              onRemove={removeAsset}
            />
          ))}
          <section className="space-y-2 rounded-2xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 p-3">
            <SwitchRow label="锁定商品主体" checked={brief.lockProduct} onChange={(lockProduct) => updateBrief({ lockProduct })} />
            <SwitchRow label="锁定品牌风格" checked={brief.lockBrandStyle} onChange={(lockBrandStyle) => updateBrief({ lockBrandStyle })} />
            <SwitchRow label="允许轻微美化" checked={brief.allowBeautify} onChange={(allowBeautify) => updateBrief({ allowBeautify })} />
            <SwitchRow label="严格结构一致" checked={brief.strictStructure} onChange={(strictStructure) => updateBrief({ strictStructure })} />
          </section>
        </aside>

        <section className="min-w-0 space-y-4">
          {ECOMMERCE_GROUPS.map((group) => {
            const items = suite.plan.filter((item) => item.group === group.id)
            return (
              <section key={group.id} className="rounded-3xl border border-gray-200 dark:border-white/[0.08] bg-gray-50/80 dark:bg-white/[0.02] p-4">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <h3 className="font-bold text-gray-950 dark:text-white">{group.label}</h3>
                    <p className="text-xs text-gray-500">{group.description}</p>
                  </div>
                  <span className="rounded-full bg-white dark:bg-gray-950 px-2.5 py-1 text-xs text-gray-500">{items.length} 项</span>
                </div>
                {items.length > 0 ? (
                  <div className="grid gap-3 2xl:grid-cols-2">
                    {items.map((item) => (
                      <PlanCard
                        key={item.id}
                        item={item}
                        tasks={tasksByItem.get(item.id) ?? []}
                        selected={selectedPlanItemId === item.id}
                        onSelect={() => setSelectedPlanItemId(item.id)}
                        onPatch={(patch) => updatePlanItem(item.id, patch)}
                        onRetry={() => void retryPlanItem(item.id)}
                        onCopyPrompt={() => void copyPrompt(item)}
                        onSetStyle={setStyle}
                        onDownload={(imageId) => void downloadImage(imageId)}
                        onEdit={setMaskEditorImageId}
                        onOpenTask={(taskId) => taskId && setDetailTaskId(taskId)}
                      />
                    ))}
                  </div>
                ) : (
                  <div className="rounded-2xl border border-dashed border-gray-200 dark:border-white/[0.08] bg-white/60 dark:bg-gray-950/50 px-4 py-10 text-center text-sm text-gray-500">
                    生成套图方案后，这里会按用途展示计划和结果。
                  </div>
                )}
              </section>
            )
          })}
        </section>

        <aside className="space-y-3">
          <section className="rounded-2xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 p-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="font-bold text-gray-950 dark:text-white">套图配置</h3>
              <button type="button" onClick={createSuite} className="text-xs font-medium text-gray-500 hover:text-gray-900 dark:hover:text-white">新建</button>
            </div>
            <div className="mt-4 space-y-3">
              <div>
                <FieldLabel>商品名称 *</FieldLabel>
                <TextInput value={brief.productName} onChange={(event) => updateBrief({ productName: event.target.value })} placeholder="例如：便携榨汁杯" />
              </div>
              <div>
                <FieldLabel>商品类目 *</FieldLabel>
                <TextInput value={brief.category} onChange={(event) => updateBrief({ category: event.target.value })} placeholder="例如：小家电 / 数码 / 美妆" />
              </div>
              <div>
                <FieldLabel>核心卖点 *</FieldLabel>
                <TextInput value={brief.sellingPoints.join('、')} onChange={(event) => updateBrief({ sellingPoints: parseTags(event.target.value) })} placeholder="便携、低噪音、易清洗" />
              </div>
              <div>
                <FieldLabel>目标人群</FieldLabel>
                <TextInput value={brief.targetAudience} onChange={(event) => updateBrief({ targetAudience: event.target.value })} placeholder="例如：通勤白领、宝妈、户外人群" />
              </div>
              <div>
                <FieldLabel>目标平台</FieldLabel>
                <div className="mt-1 flex flex-wrap gap-2">
                  {ECOMMERCE_PLATFORMS.map((platform) => {
                    const checked = brief.targetPlatforms.includes(platform)
                    return (
                      <button
                        key={platform}
                        type="button"
                        onClick={() => updateBrief({
                          targetPlatforms: checked
                            ? brief.targetPlatforms.filter((item) => item !== platform)
                            : [...brief.targetPlatforms, platform],
                        })}
                        className={`rounded-full border px-2.5 py-1 text-xs ${checked ? 'border-gray-900 bg-gray-900 text-white dark:border-white dark:bg-white dark:text-gray-950' : 'border-gray-200 dark:border-white/[0.08] text-gray-500'}`}
                      >
                        {platform}
                      </button>
                    )
                  })}
                </div>
              </div>
              <div>
                <FieldLabel>风格模板</FieldLabel>
                <SelectInput value={brief.stylePresetId} onChange={(event) => updateBrief({ stylePresetId: event.target.value })}>
                  {STYLE_PRESETS.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                </SelectInput>
                <p className="mt-1 text-xs text-gray-500">{style.description}</p>
              </div>
              <div>
                <FieldLabel>套图模板</FieldLabel>
                <SelectInput value={brief.suiteTemplateId} onChange={(event) => updateBrief({ suiteTemplateId: event.target.value })}>
                  {SUITE_TEMPLATES.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                </SelectInput>
                <p className="mt-1 text-xs text-gray-500">{template.description}</p>
              </div>
              <div>
                <FieldLabel>尺寸预设</FieldLabel>
                <SelectInput value={brief.sizePreset} onChange={(event) => updateBrief({ sizePreset: event.target.value })}>
                  {SIZE_PRESETS.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
                </SelectInput>
                <p className="mt-1 text-xs text-gray-500">{sizePreset.ratio} · {sizePreset.size}</p>
              </div>
              <div>
                <FieldLabel>每组生成项数量</FieldLabel>
                <div className="mt-1 grid grid-cols-2 gap-2">
                  {ECOMMERCE_GROUPS.map((group) => (
                    <label key={group.id} className="rounded-xl border border-gray-200 dark:border-white/[0.08] px-3 py-2 text-xs text-gray-500">
                      {group.label}
                      <input
                        type="number"
                        min={0}
                        max={12}
                        value={brief.counts[group.id]}
                        onChange={(event) => updateBrief({ counts: { ...brief.counts, [group.id]: Math.max(0, Math.min(12, Number(event.target.value) || 0)) } as EcommerceBrief['counts'] })}
                        className="mt-1 w-full bg-transparent text-sm font-semibold text-gray-900 dark:text-white outline-none"
                      />
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <FieldLabel>禁忌内容</FieldLabel>
                <TextArea rows={2} value={brief.negativePrompt ?? ''} onChange={(event) => updateBrief({ negativePrompt: event.target.value })} placeholder="不要生成竞品品牌，不夸张功效..." />
              </div>
            </div>
          </section>

          <section className="rounded-2xl border border-gray-200 dark:border-white/[0.08] bg-white dark:bg-gray-950 p-4">
            <FieldLabel>补充宣传需求</FieldLabel>
            <TextArea rows={4} value={brief.extraPrompt} onChange={(event) => updateBrief({ extraPrompt: event.target.value })} placeholder="例如：更适合小红书封面、减少文字、突出夏季清爽感..." />
            <div className="mt-3 grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => void generatePlan()}
                disabled={planStatus === 'planning' || isGenerating}
                className="rounded-xl bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45 dark:bg-white dark:text-gray-950"
              >
                {planStatus === 'planning' ? '规划中...' : '生成套图方案'}
              </button>
              <button
                type="button"
                onClick={() => void generateSuite()}
                disabled={suite.plan.length === 0 || isGenerating}
                className="rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45"
              >
                {isGenerating ? `生成中 ${generationQueue.length}` : '开始生成'}
              </button>
            </div>
            <div className="mt-2 grid grid-cols-2 gap-2">
              <button type="button" disabled className="rounded-xl border border-gray-200 dark:border-white/[0.08] px-3 py-2 text-xs text-gray-400 disabled:cursor-not-allowed" title="即将支持">全部下载 ZIP</button>
              <button type="button" disabled className="rounded-xl border border-gray-200 dark:border-white/[0.08] px-3 py-2 text-xs text-gray-400 disabled:cursor-not-allowed" title="即将支持">保存为模板</button>
            </div>
          </section>
        </aside>
      </div>
    </main>
  )
}
