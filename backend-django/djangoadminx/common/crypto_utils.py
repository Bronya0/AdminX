"""常用加解密和哈希工具

提供 SM4、AES 的常用模式加解密，以及 MD5、SHA256 哈希计算。
所有接口同时支持 bytes 和 base64 字符串两种形式。

依赖:
  - gmssl (SM4)
  - cryptography (AES)
  - hashlib (MD5, SHA256, 标准库)

使用示例:
  >>> from djangoadminx.common.crypto_utils import aes, sm4, hash
  >>> ct = aes.gcm_encrypt_b64(key_b64, nonce_b64, plaintext)
  >>> pt = aes.gcm_decrypt_b64(key_b64, nonce_b64, ct, tag_b64)
  >>> h = hash.sha256("hello")
"""

import base64
import hashlib
import os

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives import padding as _padding
from gmssl import sm4 as _sm4

# ──────────────────────────────────────────────
#  内部辅助
# ──────────────────────────────────────────────

_KB = 1024


def _check_key(key: bytes, *sizes: int):
    if len(key) not in sizes:
        raise ValueError(f"密钥长度必须是 {'/'.join(str(s) for s in sizes)} 字节，当前为 {len(key)} 字节")


def _check_iv(iv: bytes, size: int):
    if len(iv) != size:
        raise ValueError(f"IV/Nonce 长度必须是 {size} 字节，当前为 {len(iv)} 字节")


# ──────────────────────────────────────────────
#  SM4 — 国密对称加密
#  - 密钥固定 16 字节 (128 bit)
#  - 支持 ECB / CBC 模式
#  - gmssl 库自带 PKCS7 padding
# ──────────────────────────────────────────────

class sm4:
    """SM4 加解密 (ECB / CBC)"""

    @staticmethod
    def _new_cipher(key: bytes, mode: int, iv: bytes | None = None) -> _sm4.CryptSM4:
        _check_key(key, 16)
        c = _sm4.CryptSM4()
        c.set_key(key, mode)
        return c

    # -- ECB --

    @staticmethod
    def ecb_encrypt(key: bytes, data: bytes) -> bytes:
        """SM4-ECB 加密, 返回密文字节"""
        c = sm4._new_cipher(key, _sm4.SM4_ENCRYPT)
        return c.crypt_ecb(data)

    @staticmethod
    def ecb_decrypt(key: bytes, data: bytes) -> bytes:
        """SM4-ECB 解密, 返回明文字节"""
        c = sm4._new_cipher(key, _sm4.SM4_DECRYPT)
        return c.crypt_ecb(data)

    @staticmethod
    def ecb_encrypt_b64(key_b64: str, data_b64: str) -> str:
        """SM4-ECB 加密, 输入输出均为 base64 字符串"""
        return base64.b64encode(sm4.ecb_encrypt(
            base64.b64decode(key_b64), base64.b64decode(data_b64)
        )).decode()

    @staticmethod
    def ecb_decrypt_b64(key_b64: str, data_b64: str) -> str:
        """SM4-ECB 解密, 输入输出均为 base64 字符串"""
        return base64.b64encode(sm4.ecb_decrypt(
            base64.b64decode(key_b64), base64.b64decode(data_b64)
        )).decode()

    # -- CBC --

    @staticmethod
    def cbc_encrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
        """SM4-CBC 加密, 返回密文字节 (iv 16 字节)"""
        _check_key(key, 16)
        _check_iv(iv, 16)
        c = _sm4.CryptSM4()
        c.set_key(key, _sm4.SM4_ENCRYPT)
        return c.crypt_cbc(iv, data)

    @staticmethod
    def cbc_decrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
        """SM4-CBC 解密, 返回明文字节 (iv 16 字节)"""
        _check_key(key, 16)
        _check_iv(iv, 16)
        c = _sm4.CryptSM4()
        c.set_key(key, _sm4.SM4_DECRYPT)
        return c.crypt_cbc(iv, data)

    @staticmethod
    def cbc_encrypt_b64(key_b64: str, iv_b64: str, data_b64: str) -> str:
        return base64.b64encode(sm4.cbc_encrypt(
            base64.b64decode(key_b64), base64.b64decode(iv_b64), base64.b64decode(data_b64)
        )).decode()

    @staticmethod
    def cbc_decrypt_b64(key_b64: str, iv_b64: str, data_b64: str) -> str:
        return base64.b64encode(sm4.cbc_decrypt(
            base64.b64decode(key_b64), base64.b64decode(iv_b64), base64.b64decode(data_b64)
        )).decode()

    # -- 工具 --

    @staticmethod
    def generate_key() -> bytes:
        """生成随机 16 字节 SM4 密钥"""
        return os.urandom(16)

    @staticmethod
    def generate_iv() -> bytes:
        """生成随机 16 字节 IV"""
        return os.urandom(16)


# ──────────────────────────────────────────────
#  AES — 高级加密标准
#  - 密钥支持 16 / 24 / 32 字节 (AES-128/192/256)
#  - 支持 CBC / GCM 模式
#  - PKCS7 padding 由 cryptography 自动处理
# ──────────────────────────────────────────────

class aes:
    """AES 加解密 (CBC / GCM)"""

    @staticmethod
    def _check_key(key: bytes):
        _check_key(key, 16, 24, 32)

    # -- CBC --

    @staticmethod
    def _pad(data: bytes, block_size: int = 16) -> bytes:
        padder = _padding.PKCS7(block_size * 8).padder()
        return padder.update(data) + padder.finalize()

    @staticmethod
    def _unpad(data: bytes, block_size: int = 16) -> bytes:
        unpadder = _padding.PKCS7(block_size * 8).unpadder()
        return unpadder.update(data) + unpadder.finalize()

    @staticmethod
    def cbc_encrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
        """AES-CBC 加密, 返回密文字节 (iv 16 字节)"""
        aes._check_key(key)
        _check_iv(iv, 16)
        cipher = Cipher(algorithms.AES(key), modes.CBC(iv))
        encryptor = cipher.encryptor()
        return encryptor.update(aes._pad(data)) + encryptor.finalize()

    @staticmethod
    def cbc_decrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
        """AES-CBC 解密, 返回明文字节 (iv 16 字节)"""
        aes._check_key(key)
        _check_iv(iv, 16)
        cipher = Cipher(algorithms.AES(key), modes.CBC(iv))
        decryptor = cipher.decryptor()
        return aes._unpad(decryptor.update(data) + decryptor.finalize())

    @staticmethod
    def cbc_encrypt_b64(key_b64: str, iv_b64: str, data_b64: str) -> str:
        return base64.b64encode(aes.cbc_encrypt(
            base64.b64decode(key_b64), base64.b64decode(iv_b64), base64.b64decode(data_b64)
        )).decode()

    @staticmethod
    def cbc_decrypt_b64(key_b64: str, iv_b64: str, data_b64: str) -> str:
        return base64.b64encode(aes.cbc_decrypt(
            base64.b64decode(key_b64), base64.b64decode(iv_b64), base64.b64decode(data_b64)
        )).decode()

    # -- GCM (认证加密) --

    @staticmethod
    def gcm_encrypt(key: bytes, nonce: bytes, data: bytes, aad: bytes = b"") -> tuple[bytes, bytes]:
        """AES-GCM 加密, 返回 (密文, 认证标签)
        nonce 推荐 12 字节 (96 bit), aad 为附加认证数据 (可选)
        """
        aes._check_key(key)
        _check_iv(nonce, 12)
        cipher = Cipher(algorithms.AES(key), modes.GCM(nonce))
        encryptor = cipher.encryptor()
        encryptor.authenticate_additional_data(aad)
        ct = encryptor.update(data) + encryptor.finalize()
        return ct, encryptor.tag

    @staticmethod
    def gcm_decrypt(key: bytes, nonce: bytes, data: bytes, tag: bytes, aad: bytes = b"") -> bytes:
        """AES-GCM 解密, 返回明文字节
        认证失败时抛出 InvalidTag 异常
        """
        aes._check_key(key)
        _check_iv(nonce, 12)
        cipher = Cipher(algorithms.AES(key), modes.GCM(nonce, tag))
        decryptor = cipher.decryptor()
        decryptor.authenticate_additional_data(aad)
        return decryptor.update(data) + decryptor.finalize()

    @staticmethod
    def gcm_encrypt_b64(key_b64: str, nonce_b64: str, data_b64: str, aad_b64: str = "") -> dict:
        """AES-GCM 加密, 输入输出 base64 字符串
        返回 {"ciphertext": str, "tag": str}
        """
        ct, tag = aes.gcm_encrypt(
            base64.b64decode(key_b64), base64.b64decode(nonce_b64),
            base64.b64decode(data_b64), base64.b64decode(aad_b64) if aad_b64 else b"",
        )
        return {
            "ciphertext": base64.b64encode(ct).decode(),
            "tag": base64.b64encode(tag).decode(),
        }

    @staticmethod
    def gcm_decrypt_b64(key_b64: str, nonce_b64: str, ciphertext_b64: str, tag_b64: str, aad_b64: str = "") -> str:
        """AES-GCM 解密, 输入输出 base64 字符串"""
        pt = aes.gcm_decrypt(
            base64.b64decode(key_b64), base64.b64decode(nonce_b64),
            base64.b64decode(ciphertext_b64), base64.b64decode(tag_b64),
            base64.b64decode(aad_b64) if aad_b64 else b"",
        )
        return base64.b64encode(pt).decode()

    # -- 工具 --

    @staticmethod
    def generate_key(size: int = 32) -> bytes:
        """生成随机 AES 密钥, size=16/24/32 对应 AES-128/192/256"""
        if size not in (16, 24, 32):
            raise ValueError("size 必须是 16 (AES-128), 24 (AES-192), 32 (AES-256)")
        return os.urandom(size)

    @staticmethod
    def generate_nonce() -> bytes:
        """生成随机 12 字节 nonce (GCM 推荐)"""
        return os.urandom(12)

    @staticmethod
    def generate_iv() -> bytes:
        """生成随机 16 字节 IV (CBC 使用)"""
        return os.urandom(16)


# ──────────────────────────────────────────────
#  Hash — 哈希/摘要
# ──────────────────────────────────────────────

class hash:
    """MD5 / SHA256 哈希"""

    @staticmethod
    def md5(data: bytes, upper: bool = False) -> str:
        """MD5 摘要, 返回 32 位十六进制小写字符串"""
        return hashlib.md5(data).hexdigest().upper() if upper else hashlib.md5(data).hexdigest()

    @staticmethod
    def sha256(data: bytes, upper: bool = False) -> str:
        """SHA-256 摘要, 返回 64 位十六进制小写字符串"""
        return hashlib.sha256(data).hexdigest().upper() if upper else hashlib.sha256(data).hexdigest()

    @staticmethod
    def md5_file(path: str) -> str:
        """计算文件的 MD5 (适合大文件, 流式读取)"""
        h = hashlib.md5()
        with open(path, "rb") as f:
            for chunk in iter(lambda: f.read(64 * _KB), b""):
                h.update(chunk)
        return h.hexdigest()

    @staticmethod
    def sha256_file(path: str) -> str:
        """计算文件的 SHA-256 (流式读取)"""
        h = hashlib.sha256()
        with open(path, "rb") as f:
            for chunk in iter(lambda: f.read(64 * _KB), b""):
                h.update(chunk)
        return h.hexdigest()

    @staticmethod
    def hmac_sha256(key: bytes, data: bytes) -> str:
        """HMAC-SHA256, 返回十六进制字符串"""
        import hmac as _hmac
        return _hmac.new(key, data, hashlib.sha256).hexdigest()