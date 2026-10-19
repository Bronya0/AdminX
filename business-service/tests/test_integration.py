"""Run: python -m unittest discover -s tests -p 'test_*.py' (from business-service)."""
import unittest
from unittest.mock import AsyncMock, patch

import httpx
import main
import auth
from routers import posts


class BusinessIntegrationTests(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        posts._db.clear()
        posts._counter = 0
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=main.app), base_url="http://business.test")

    async def asyncTearDown(self):
        await self.client.aclose()

    async def test_bootstrap_config_and_embed_policy(self):
        response = await self.client.get("/")
        self.assertEqual(response.status_code, 200)
        self.assertIn('"platformOrigins"', response.text)
        self.assertNotIn('__ADMINX_CONFIG__', response.text)
        self.assertIn('frame-ancestors', response.headers['content-security-policy'])
        self.assertEqual(response.headers['cache-control'], 'no-store')
        self.assertEqual((await self.client.get('/static/adminx-bridge.js')).status_code, 200)

    async def test_auth_failure_and_unavailable_are_distinct(self):
        self.assertEqual((await self.client.get('/api/v1/posts/')).status_code, 401)
        with patch.object(main, 'introspect_token', AsyncMock(return_value=None)):
            self.assertEqual((await self.client.get('/api/v1/posts/', headers={'Authorization':'Bearer bad'})).status_code, 401)
        with patch.object(main, 'introspect_token', AsyncMock(side_effect=RuntimeError('unavailable'))):
            self.assertEqual((await self.client.get('/api/v1/posts/', headers={'Authorization':'Bearer access'})).status_code, 502)

    async def test_crud_keeps_user_data_isolated(self):
        async def identity(token):
            # 同名但 ID 不同，模拟旧账号删除后创建同名账号。
            return {'user_id': token, 'username': 'same-name', 'exp': 9999999999}
        with patch.object(main, 'introspect_token', identity):
            alice = {'Authorization': 'Bearer alice'}
            bob = {'Authorization': 'Bearer bob'}
            response = await self.client.post('/api/v1/posts/', headers=alice, json={'title':'<script>alert(1)</script>', 'content':'body'})
            self.assertEqual(response.status_code, 201)
            post_id = response.json()['data']['id']
            self.assertEqual((await self.client.get('/api/v1/posts/', headers=bob)).json()['data']['count'], 0)
            self.assertEqual((await self.client.delete(f'/api/v1/posts/{post_id}', headers=bob)).status_code, 404)
            response = await self.client.put(f'/api/v1/posts/{post_id}', headers=alice, json={'title':'updated','content':'safe','status':'published'})
            self.assertEqual(response.json()['data']['title'], 'updated')
            self.assertEqual((await self.client.delete(f'/api/v1/posts/{post_id}', headers=alice)).status_code, 200)

    async def test_introspection_network_failure_does_not_masquerade_as_expired_token(self):
        with patch.object(auth.httpx, 'AsyncClient') as factory:
            client = factory.return_value.__aenter__.return_value
            client.post = AsyncMock(side_effect=httpx.ConnectError('offline'))
            with self.assertRaises(RuntimeError):
                await auth.introspect_token('access')

    async def test_manual_registration_requires_component_secret(self):
        self.assertEqual((await self.client.post('/api/v1/register/')).status_code, 403)
