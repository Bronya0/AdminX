"""
Spyne 兼容垫片 — Python 3.13+ 适配

Spyne 2.14.0 捆绑的 six 1.14.0 中 _SixMetaPathImporter 仅实现了
已弃用的 find_module/load_module 协议，Python 3.12+ 中已失效。

本模块在 spyne 正式导入前，单独加载其捆绑的 six.py，在其
_SixMetaPathImporter 上安装 find_spec 方法，使 moves 子模块导入
在 Python 3.13+ 下能正常工作。

使用方法：在 manage.py、wsgi.py、asgi.py 入口文件最顶部导入本模块。
"""

import importlib.abc
import importlib.machinery
import importlib.util
import os
import sys
import types


class _StubLoader(importlib.abc.Loader):
    """虚拟 Loader — 模块已被打补丁塞入 sys.modules，无需真正执行。"""
    def create_module(self, spec):
        return sys.modules.get(spec.name)

    def exec_module(self, module):
        pass


def apply():
    """在 spyne 加载前，给其捆绑的 _SixMetaPathImporter 打上 find_spec 补丁。"""
    if 'spyne' in sys.modules:
        return

    # ── 1. 定位 spyne.util.six 的路径 ──
    spec = importlib.util.find_spec('spyne')
    if spec is None or not spec.submodule_search_locations:
        return

    spyne_path = spec.submodule_search_locations[0]
    six_path = os.path.join(spyne_path, 'util', 'six.py')
    if not os.path.isfile(six_path):
        return

    # ── 2. 创建 spyne / spyne.util 包占位 ──
    for name, pkg_path in [
        ('spyne', spyne_path),
        ('spyne.util', os.path.join(spyne_path, 'util')),
    ]:
        if name not in sys.modules:
            mod = types.ModuleType(name)
            mod.__path__ = [pkg_path]
            mod.__package__ = name
            sys.modules[name] = mod

    # ── 3. 单独加载 spyne.util.six ──
    loader = importlib.machinery.SourceFileLoader('spyne.util.six', six_path)
    six_mod = loader.load_module('spyne.util.six')

    # ── 3b. 清理占位包，让后续 "import spyne" 能正常加载真实包 ──
    # 注意：只移除空的占位包，spyne.util.six 保留在 sys.modules 中
    for name in ('spyne.util', 'spyne'):
        if name in sys.modules and not hasattr(sys.modules[name], '__file__'):
            del sys.modules[name]

    # ── 4. 给 _SixMetaPathImporter 添加 find_spec ──
    ImporterCls = getattr(six_mod, '_SixMetaPathImporter', None)
    if ImporterCls is None:
        return

    def _find_spec(self, fullname, path, target=None):
        if fullname not in self.known_modules:
            return None

        mod = self.known_modules[fullname]

        # 如果是 MovedModule，先解析出真正的模块对象
        if isinstance(mod, six_mod.MovedModule):
            try:
                mod = mod._resolve()
            except Exception:
                return None

        # 放入 sys.modules，确保后续导入直接命中
        sys.modules[fullname] = mod

        return importlib.machinery.ModuleSpec(
            fullname,
            loader=_StubLoader(),
            origin=f'spyne_compat:{fullname}',
        )

    ImporterCls.find_spec = _find_spec

    # ── 5. 预注册已知的 moves 子模块，加快后续查找 ──
    for key in list(six_mod._importer.known_modules.keys()):
        if key.startswith('spyne.util.six.moves.'):
            mod = six_mod._importer.known_modules[key]
            if isinstance(mod, six_mod.MovedModule):
                try:
                    sys.modules[key] = mod._resolve()
                except Exception:
                    pass

    # 注册 six.moves 自身
    moves_key = 'spyne.util.six.moves'
    if moves_key in six_mod._importer.known_modules:
        moves_mod = six_mod._importer.known_modules[moves_key]
        if not isinstance(moves_mod, six_mod.MovedModule):
            sys.modules[moves_key] = moves_mod


apply()
