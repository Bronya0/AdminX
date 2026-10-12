// 触发浏览器下载的公共实现：把 Blob 存成文件。
// revokeObjectURL 必须延后到下一个事件循环：同步释放会让部分浏览器
// （Firefox/Safari）在下载真正开始前就拿到失效的地址，表现为点了没反应。
export const downloadBlob = (blob: Blob, filename: string) => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 0)
}
