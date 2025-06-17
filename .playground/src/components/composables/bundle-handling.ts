import { ref, UnwrapNestedRefs, watch } from "vue"

import JsZip from "jszip"
import FileSaver from "file-saver"
import { getBaseOptions, getBaseRenderData, RenderTemplateDataViewModel } from "@/models/render-data-base"
import { Asset, BundleAssetsFolder } from "@/models/asset"
import {
  EnumRenderTemplateDataTemplateEngine,
  ListHtmlBundlesInfoService,
  SaveHtmlBundleService,
} from "@/swagger-client"
import { serverBaseUrl } from "@/config/server"

type IncludedFiles = {
  index?: boolean
  header?: boolean
  footer?: boolean
  options?: boolean
  assets?: boolean
  exampleModel?: boolean
}

type BundleInfo = {
  name: string
  renamedFrom?: string
}

const includedFilesAll: IncludedFiles = {
  index: true,
  header: true,
  footer: true,
  options: true,
  assets: true,
  exampleModel: true,
}

export function packBundle(data: RenderTemplateDataViewModel, includedFiles = includedFilesAll): Promise<Blob> {
  const zip = new JsZip()

  if (includedFiles.index) {
    zip.file("index.html", data.htmlTemplate ?? "")
  }
  if (includedFiles.header && data.headerHtmlTemplate) {
    zip.file("header.html", data.headerHtmlTemplate)
  }
  if (includedFiles.footer && data.footerHtmlTemplate) {
    zip.file("footer.html", data.footerHtmlTemplate)
  }
  if (includedFiles.options) {
    zip.file("options.json", JSON.stringify(data.options))
  }
  if (includedFiles.exampleModel) {
    zip.file("example-model.json", JSON.stringify(JSON.parse(data.modelStr)))
  }

  if (includedFiles.assets) {
    for (const a of data.assets) {
      zip.file(`assets/${a.name}`, a.blob)
    }
  }

  return zip.generateAsync({ type: "blob" })
}

export function useBundleHandling(reactiveRenderTemplateDataViewModel: RenderTemplateDataViewModel, settings: UnwrapNestedRefs<{
  autoRenderDelay: number;
  serverUrl: string;
  secret: string
}>) {
  const loadBundle = async (
    bundleBufferSrc: ArrayBuffer,
    target: RenderTemplateDataViewModel = getBaseRenderData(),
  ): Promise<RenderTemplateDataViewModel> => {
    const zip = await JsZip.loadAsync(bundleBufferSrc)

    const indexStr = await zip.files["index.html"]?.async("string")
    const headerStr = await zip.files["header.html"]?.async("string")
    const footerStr = await zip.files["footer.html"]?.async("string")
    const optionsStr = await zip.files["options.json"]?.async("string")
    const exampleModelStr = (await zip.files["example-model.json"]?.async("string")) ?? "{}"

    const assetFolderPath = BundleAssetsFolder + "/"
    const assets: Asset[] = []
    for (const fKey in zip.files) {
      if (fKey.startsWith(assetFolderPath) && fKey !== assetFolderPath) {
        assets.push({
          name: fKey.substring(assetFolderPath.length),
          blob: await zip.files[fKey].async("blob"),
        })
      }
    }

    if (indexStr) {
      target.htmlTemplate = indexStr
    }
    if (headerStr) {
      target.headerHtmlTemplate = headerStr
    }
    if (footerStr) {
      target.footerHtmlTemplate = footerStr
    }
    if (optionsStr) {
      target.options = { ...getBaseOptions(), ...JSON.parse(optionsStr) }
    }
    if (exampleModelStr) {
      const model = JSON.parse(exampleModelStr)
      target.model = model
      target.modelStr = JSON.stringify(model, null, 2)
    }

    if (assets.length > 0) {
      target.assets = assets
    }

    return target
  }

  const downloadBundle = async (
    only?: "documentWithoutHeaderAndFooter" | "onlyBody" | "onlyHeader" | "onlyFooter" | "onlyOptions" | "onlyAssets",
  ) => {
    const includedFiles = ((): IncludedFiles | undefined => {
      switch (only) {
        case "documentWithoutHeaderAndFooter":
          return { index: true, options: true, assets: true }
        case "onlyBody":
          return { index: true }
        case "onlyHeader":
          return { header: true }
        case "onlyFooter":
          return { footer: true }
        case "onlyOptions":
          return { options: true }
        case "onlyAssets":
          return { assets: true }
        case undefined:
          return undefined
      }
    })()

    const zip = await packBundle(reactiveRenderTemplateDataViewModel, includedFiles)

    FileSaver.saveAs(zip, "pdf-turtle-bundle.zip")
  }

  const loadBundlesInfo = async (): Promise<BundleInfo[]> => {
    try {
      const res = await ListHtmlBundlesInfoService.htmlBundle({
        loading: false,
        responseType: "json",
        baseURL: settings.serverUrl || undefined,
        headers: !settings.secret ? undefined : { Authorization: `Bearer ${settings.secret}` },
      })
      return res.Items
    } catch (e) {
      bundleError.value = `Error loading bundles info: ${e}`
    }
    return []
  }

  const getBundle = async (name: string): Promise<void> => {
    try {
      const baseUrl = settings.serverUrl || serverBaseUrl
      const res = await fetch(`${baseUrl}/api/html-bundle/${encodeURIComponent(name)}`, {
        headers: !settings.secret ? undefined : { Authorization: `Bearer ${settings.secret}` },
      })
      if (!res.ok) {
        bundleError.value = `Error loading bundle: ${res.statusText}`
        return
      }
      const data = await res.formData()
      const bundle = data.get("bundle") as File

      await loadBundle(await bundle.arrayBuffer(), reactiveRenderTemplateDataViewModel)
      const engine = data.get("templateEngine") as keyof typeof EnumRenderTemplateDataTemplateEngine
      reactiveRenderTemplateDataViewModel.templateEngine = EnumRenderTemplateDataTemplateEngine[engine]

      currentBundle.value = { name: data.get("name") as string }
      dirty.value = false
    } catch (e) {
      bundleError.value = `Error loading bundle: ${e}`
    }
  }

  const storeBundle = async (name: string, renameFrom?: string): Promise<string> => {
    try {
      const bundle = await packBundle(reactiveRenderTemplateDataViewModel)
      const res = await SaveHtmlBundleService.htmlBundle(
        {
          bundle,
          name,
          templateEngine: reactiveRenderTemplateDataViewModel.templateEngine ?? 'golang',
          renameFrom,
        },
        {
          loading: true,
          responseType: "json",
          baseURL: settings.serverUrl || undefined,
          headers: !settings.secret ? undefined : { Authorization: `Bearer ${settings.secret}` },
        },
      )
      currentBundle.value = { name }
      dirty.value = false
      return res.id
    } catch (e) {
      bundleError.value = `Error saving bundle: ${e}`
    }
    return ""
  }

  const cleanLocalStorageBundle = () => {
    localStorage.removeItem("currentBundle")
  }

  const currentBundle = ref<BundleInfo | null>(null)
  const dirty = ref(false)
  const bundleError = ref<string | null>(null)
  const bundleFileInputModel = ref<File | null>(null)
  watch(bundleFileInputModel, async (b) => {
    if (b) {
      await loadBundle(await b.arrayBuffer(), reactiveRenderTemplateDataViewModel)
      currentBundle.value = null
      cleanLocalStorageBundle()
    }

    bundleFileInputModel.value = null
  })

  watch(reactiveRenderTemplateDataViewModel, () => {
    dirty.value = true
  })

  window.onbeforeunload = (e) => {
    if (currentBundle.value?.name) {
      localStorage.setItem("currentBundle", currentBundle.value.name)
    }
    if (dirty.value) {
      e.preventDefault()
      e.returnValue = "there are unsaved data, are you sure you want to leave?"
    }
  }

  const savedBundleId = localStorage.getItem("currentBundle")
  if (savedBundleId) {
    getBundle(savedBundleId).catch(console.error)
  }

  return {
    getBundle,
    storeBundle,
    loadBundlesInfo,
    loadBundle,
    downloadBundle,
    cleanLocalStorageBundle,
    bundleError,
    currentBundle,
    bundleFileInputModel,
  }
}
