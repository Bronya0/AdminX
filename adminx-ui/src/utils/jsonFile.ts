// 导入导出 JSON 文件的浏览器端工具（字典配置、主题设置共用）

/** 把数据保存为 `<name>-YYYY-MM-DD.json` 并触发浏览器下载 */
export const downloadJsonFile = (name: string, data: unknown) => {
  const date = new Date().toISOString().slice(0, 10)
  const url = URL.createObjectURL(new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `${name}-${date}.json`
  link.click()
  // 延后释放，避免部分浏览器在下载开始前就失效
  setTimeout(() => URL.revokeObjectURL(url), 0)
}

/** 读取并解析 JSON 文件；内容不是合法 JSON 时抛出可直接展示的错误 */
export const readJsonFile = async (file: File): Promise<unknown> => {
  try {
    return JSON.parse(await file.text())
  } catch {
    throw new Error('文件不是有效的 JSON 格式')
  }
}
