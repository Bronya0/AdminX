"""crypto_utils — 加解密工具"""

from djangoadminx.common.crypto_utils import sm4, aes, hash


class TestSm4:
    def test_ecb_encrypt_decrypt(self):
        key = sm4.generate_key()
        data = b"hello world"
        ct = sm4.ecb_encrypt(key, data)
        pt = sm4.ecb_decrypt(key, ct)
        assert pt == data

    def test_cbc_encrypt_decrypt(self):
        key = sm4.generate_key()
        iv = sm4.generate_iv()
        data = b"cbc test data"
        ct = sm4.cbc_encrypt(key, iv, data)
        pt = sm4.cbc_decrypt(key, iv, ct)
        assert pt == data

    def test_ecb_b64(self):
        key_b64 = sm4.ecb_encrypt_b64.__wrapped__(None, None)  # skip
        # 直接用静态方法
        import base64
        key = sm4.generate_key()
        data = b"b64test"
        ct_b64 = sm4.ecb_encrypt_b64(base64.b64encode(key).decode(), base64.b64encode(data).decode())
        pt_b64 = sm4.ecb_decrypt_b64(base64.b64encode(key).decode(), ct_b64)
        assert pt_b64 == base64.b64encode(data).decode()


class TestAes:
    def test_cbc_encrypt_decrypt(self):
        key = aes.generate_key(16)
        iv = aes.generate_iv()
        data = b"aes cbc test"
        ct = aes.cbc_encrypt(key, iv, data)
        pt = aes.cbc_decrypt(key, iv, ct)
        assert pt == data

    def test_gcm_encrypt_decrypt(self):
        key = aes.generate_key(16)
        nonce = aes.generate_nonce()
        data = b"aes gcm test"
        ct, tag = aes.gcm_encrypt(key, nonce, data)
        pt = aes.gcm_decrypt(key, nonce, ct, tag)
        assert pt == data

    def test_gcm_with_aad(self):
        key = aes.generate_key(16)
        nonce = aes.generate_nonce()
        ct, tag = aes.gcm_encrypt(key, nonce, b"data", b"aad")
        pt = aes.gcm_decrypt(key, nonce, ct, tag, b"aad")
        assert pt == b"data"

    def test_gcm_b64(self):
        import base64
        key = aes.generate_key(16)
        nonce = aes.generate_nonce()
        result = aes.gcm_encrypt_b64(
            base64.b64encode(key).decode(),
            base64.b64encode(nonce).decode(),
            base64.b64encode(b"b64").decode(),
        )
        pt = aes.gcm_decrypt_b64(
            base64.b64encode(key).decode(),
            base64.b64encode(nonce).decode(),
            result["ciphertext"], result["tag"],
        )
        assert pt == base64.b64encode(b"b64").decode()


class TestHash:
    def test_md5(self):
        h = hash.md5(b"hello")
        assert len(h) == 32

    def test_sha256(self):
        h = hash.sha256(b"hello")
        assert len(h) == 64

    def test_hmac_sha256(self):
        h = hash.hmac_sha256(b"key", b"data")
        assert len(h) == 64

    def test_md5_upper(self):
        h = hash.md5(b"hello", upper=True)
        assert h.isupper()

    def test_md5_file(self):
        import tempfile, os
        with tempfile.NamedTemporaryFile(delete=False) as f:
            f.write(b"file content")
            path = f.name
        try:
            h = hash.md5_file(path)
            assert len(h) == 32
        finally:
            os.unlink(path)

    def test_sha256_file(self):
        import tempfile, os
        with tempfile.NamedTemporaryFile(delete=False) as f:
            f.write(b"file data")
            path = f.name
        try:
            h = hash.sha256_file(path)
            assert len(h) == 64
        finally:
            os.unlink(path)
