const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright-core');
const assert = require('node:assert/strict');
(async () => {
 const browser = await chromium.launch({ executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE, headless:true });
 try {
 const context = await browser.newContext();
 let refreshes = 0;
 const jwt = n => 'header.' + Buffer.from(JSON.stringify({exp:Math.floor(Date.now()/1000)+3600,n})).toString('base64url') + '.signature';
 let latest = jwt(0);
 const invalid = new Set([latest]);
 let posts = [{id:1,title:'<img src=x onerror="window.xss=true">',content:'示例内容',status:'draft',created_at:new Date().toISOString()}];
 const menu = (id,code,name,path,type='menu') => ({id,code,name,path,menu_type:type,is_active:true,is_visible:true,sort_order:id,depth:1,icon:'',component:'',permission_code:'',parent_id:null,allowed_paths:[]});
 const menus = [menu(1,'dashboard','仪表盘','/dashboard'),menu(2,'demo','文章管理','http://localhost:18001/','iframe'),menu(3,'demo:newtab','业务新窗口','http://localhost:18001/')];
 await context.addInitScript(token => {
   if (location.origin === 'http://localhost:5179' && !localStorage.getItem('user-store')) {
    localStorage.setItem('user-store',JSON.stringify({token})); sessionStorage.setItem('adminx_refresh_token','refresh-0');
   }
 }, latest);
 await context.route('**/adminx/api/v1/**',async route => {
  const path = new URL(route.request().url()).pathname;
  let data = {};
  if(path.endsWith('/accounts/users/me/')) data = {user:{id:'u1',username:'demo-user',roles:[],is_superuser:true},permissions:[],menus};
  else if(path.endsWith('/accounts/refresh/')) {refreshes++;latest=jwt(refreshes);data={access:latest,refresh:'refresh-'+refreshes};await new Promise(r=>setTimeout(r,50));}
  else if(path.endsWith('/common/theme/')) data={isDark:false,primaryColor:'#1677ff',fontSize:14,borderRadius:6,layout:'side'};
  else if(path.endsWith('/common/site-info/')) data={site_name:'AdminX',idle_timeout:30};
  else if(path.endsWith('/notification/messages/')) data={results:[],count:0};
  else if(path.endsWith('/unread_count/')) data={count:0};
  await route.fulfill({json:{code:200,msg:'ok',data}});
 });
 await context.route('**/api/v1/posts/**',async route => {
  const req=route.request();const token=(req.headers().authorization||'').replace('Bearer ','');
  if(invalid.has(token))return route.fulfill({status:401,json:{code:401,msg:'expired'}});
  const url=new URL(req.url());const id=Number(url.pathname.split('/').filter(Boolean).at(-1));
  let data;
  if(req.method()==='DELETE'){posts=posts.filter(p=>p.id!==id);data=null;}
  else if(req.method()==='PUT'){Object.assign(posts.find(p=>p.id===id),req.postDataJSON());data=posts.find(p=>p.id===id);}
  else if(req.method()==='POST'){data={...req.postDataJSON(),id:2,created_at:new Date().toISOString()};posts.push(data);}
  else data=id?posts.find(p=>p.id===id):{results:posts,count:posts.length};
  await route.fulfill({json:{code:200,msg:'ok',data}});
 });
 const page=await context.newPage();
 const errors=[];page.on('pageerror',e=>errors.push(e.message));page.on('console',m=>{if(m.type()==='error')console.log('console',m.text());});
 const url='http://localhost:5179/adminx/iframe?url='+encodeURIComponent('http://localhost:18001/')+'&title=文章管理';
 await page.goto(url);
 let frame=page.frameLocator('iframe');
 try { await frame.locator('#businessPanel').waitFor({state:'visible',timeout:10000}); } catch(e){console.log('debug',page.url(),await page.locator('body').innerText(),page.frames().map(f=>f.url()),errors);for(const f of page.frames())if(f.url().includes('18001'))console.log(await f.locator('body').innerText());throw e;}
 await frame.locator('#postList strong').waitFor();
 assert.equal(refreshes,1,'initial 401 refreshed once');
 assert.equal(await frame.locator('#postList strong').textContent(),posts[0].title);
 assert.equal(await page.frames().find(f=>f.url().startsWith('http://localhost:18001')).evaluate(()=>Boolean(window.xss)),false);
 await page.evaluate(async()=>{const m=await import('/adminx/src/stores/user.ts');m.useUserStore().updateTheme({isDark:true,primaryColor:'#00aa88',fontSize:16,borderRadius:10});});
 await page.waitForFunction(()=>document.querySelector('iframe'));
 await frame.locator('html.dark').waitFor();
 assert.equal(await frame.locator('html').evaluate(e=>e.style.getPropertyValue('--primary')),'#00aa88');
 await frame.getByRole('button',{name:'编辑',exact:true}).click();
 await frame.getByLabel('文章标题',{exact:true}).fill('修改后的标题');
 await page.waitForTimeout(100);
 await page.getByText('仪表盘',{exact:true}).first().click();
 await page.getByText('还有未保存的内容，确定离开？',{exact:true}).waitFor();
 await page.getByRole('button',{name:/取.*消/}).last().click();
 await frame.getByRole('button',{name:'保存',exact:true}).click();
 await frame.locator('#postList strong').filter({hasText:'修改后的标题'}).waitFor();
 await frame.getByRole('button',{name:'编辑',exact:true}).click();
 await page.waitForURL(/businessRoute/);
 await page.reload();
 frame=page.frameLocator('iframe');
 await frame.getByLabel('文章标题',{exact:true}).waitFor();
 assert.equal(await frame.getByLabel('文章标题',{exact:true}).inputValue(),'修改后的标题');
 await frame.getByRole('button',{name:/取.*消/}).click();
 invalid.add(latest);const before=refreshes;
 const child=page.frames().find(f=>f.url().startsWith('http://localhost:18001'));
 await child.evaluate(()=>Promise.all([window.AdminXBridge.request('api/v1/posts/'),window.AdminXBridge.request('api/v1/posts/')]));
 assert.equal(refreshes,before+1,'concurrent business requests share renewal');
 await frame.getByRole('button',{name:'删除',exact:true}).click();
 await page.getByText('确定删除此文章？',{exact:true}).waitFor();
 await page.getByRole('button',{name:/取.*消/}).last().click();
 assert.equal(posts.length,1);
 const popupPromise=page.waitForEvent('popup');
 await page.getByText('业务新窗口',{exact:true}).click();
 const popup=await popupPromise;
 await popup.locator('#businessPanel').waitFor({state:'visible'});
 await page.evaluate(async()=>{const m=await import('/adminx/src/stores/user.ts');m.useUserStore().clearToken();});
 await popup.locator('#connectionPanel').waitFor({state:'visible'});
 assert.equal(await popup.locator('#titleInput').inputValue(),'');
 assert.equal(await popup.evaluate(()=>localStorage.getItem('business_token')),null);
 assert.equal(await popup.locator('#postList').textContent(),'');
 assert.deepEqual(errors,[],'no platform JS errors');
 await popup.close();
 const direct=await context.newPage();await direct.goto('http://localhost:18001/');
 await direct.getByText('请从 AdminX 平台进入业务页面',{exact:true}).waitFor();

 console.log('Browser integration passed: handshake, theme, renewal, dirty guard, route restore, confirm, logout, direct open');
 } finally {await browser.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
