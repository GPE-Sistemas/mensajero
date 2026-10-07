'use strict';
const test = require('node:test');
const assert = require('node:assert');
const http = require('node:http');
const { Mensajero, MensajeroError } = require('./index');

function servidor(responder) {
  return new Promise((resolve) => {
    const s = http.createServer((req, res) => {
      let cuerpo = '';
      req.on('data', (c) => (cuerpo += c));
      req.on('end', () => responder(req, JSON.parse(cuerpo || '{}'), res));
    });
    s.listen(0, '127.0.0.1', () => resolve(s));
  });
}

test('manda el pedido y devuelve el id', async (t) => {
  let visto;
  const s = await servidor((req, cuerpo, res) => {
    visto = { auth: req.headers.authorization, url: req.url, cuerpo };
    res.writeHead(200, { 'Content-Type': 'application/json' }).end('{"id":"abc@mensajero.gpe.ar"}');
  });
  t.after(() => s.close());
  const m = new Mensajero({ url: `http://127.0.0.1:${s.address().port}/`, apikey: 'k' });
  const r = await m.enviar('reset-password', { para: 'a@x.com', datos: { link: 'l' } });
  assert.deepStrictEqual(r, { id: 'abc@mensajero.gpe.ar' });
  assert.strictEqual(visto.auth, 'Bearer k');
  assert.strictEqual(visto.url, '/v1/email');
  assert.deepStrictEqual(visto.cuerpo, { plantilla: 'reset-password', para: 'a@x.com', datos: { link: 'l' } });
});

test('un error del servidor llega con su status y mensaje', async (t) => {
  const s = await servidor((_req, _c, res) => res.writeHead(400).end('{"error":"falta \\"link\\""}'));
  t.after(() => s.close());
  const m = new Mensajero({ url: `http://127.0.0.1:${s.address().port}`, apikey: 'k' });
  await assert.rejects(m.enviar('x', { para: 'a@x.com', datos: {} }), (e) => e instanceof MensajeroError && e.status === 400 && e.message === 'falta "link"');
});

test('si no llega al mensajero, status 0 con la causa', async () => {
  const s = await servidor(() => {});
  const puerto = s.address().port;
  await new Promise((r) => s.close(r)); // puerto cerrado: connection refused
  const m = new Mensajero({ url: `http://127.0.0.1:${puerto}`, apikey: 'k', timeoutMs: 2000 });
  await assert.rejects(m.enviar('x', { para: 'a@x.com', datos: {} }), (e) => e instanceof MensajeroError && e.status === 0 && e.message.includes('ECONNREFUSED'));
});

async function rechazaCon(t, responder, status, opciones = {}) {
  const s = await servidor(responder);
  t.after(() => { s.closeAllConnections(); s.close(); });
  const m = new Mensajero({ url: `http://127.0.0.1:${s.address().port}`, apikey: 'k', ...opciones });
  await assert.rejects(m.enviar('x', { para: 'a@x.com', datos: {} }), (e) => e instanceof MensajeroError && e.status === status);
}

test('200 con cuerpo que no es JSON es error', (t) => rechazaCon(t, (_r, _c, res) => res.writeHead(200).end('<html>proxy</html>'), 200));
test('200 sin id es error', (t) => rechazaCon(t, (_r, _c, res) => res.writeHead(200).end('{}'), 200));
test('500 con cuerpo que no es JSON conserva el status', (t) => rechazaCon(t, (_r, _c, res) => res.writeHead(500).end('boom'), 500));
test('error con cuerpo null conserva el status', (t) => rechazaCon(t, (_r, _c, res) => res.writeHead(502).end('null'), 502));
test('timeout leyendo el cuerpo es status 0', (t) =>
  rechazaCon(t, (_r, _c, res) => res.writeHead(200, { 'Content-Type': 'application/json' }).write('{"id":'), 0, { timeoutMs: 300 }));
