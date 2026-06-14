"""common — renderer + exceptions"""

import json
from rest_framework.response import Response
from rest_framework.exceptions import PermissionDenied, ValidationError
from djangoadminx.common.renderers import StandardJsonRenderer
from djangoadminx.common.exceptions import custom_exception_handler


class TestStandardJsonRenderer:
    def test_wraps_normal(self):
        renderer = StandardJsonRenderer()
        resp = Response({"foo": "bar"}, status=200)
        ctx = {"response": resp}
        rendered = renderer.render(resp.data, renderer_context=ctx)
        data = json.loads(rendered)
        assert data["code"] == 200
        assert data["data"]["foo"] == "bar"

    def test_passthrough(self):
        renderer = StandardJsonRenderer()
        resp = Response({"code": 200, "msg": "ok", "data": {}}, status=200)
        ctx = {"response": resp}
        rendered = renderer.render(resp.data, renderer_context=ctx)
        data = json.loads(rendered)
        assert data["code"] == 200

    def test_error_empty(self):
        renderer = StandardJsonRenderer()
        resp = Response({}, status=400)
        ctx = {"response": resp}
        rendered = renderer.render(resp.data, renderer_context=ctx)
        data = json.loads(rendered)
        assert data["code"] == 400

    def test_error_with_detail(self):
        renderer = StandardJsonRenderer()
        resp = Response({"detail": "bad request"}, status=400)
        ctx = {"response": resp}
        rendered = renderer.render(resp.data, renderer_context=ctx)
        data = json.loads(rendered)
        assert data["msg"] == "bad request"


class TestExceptionHandler:
    def test_permission_denied(self):
        resp = custom_exception_handler(PermissionDenied("no perm"), {})
        assert resp is not None
        assert resp.data["code"] == 403

    def test_validation_error(self):
        resp = custom_exception_handler(ValidationError({"field": ["invalid"]}), {})
        assert resp is not None
        assert "invalid" in resp.data["msg"]
