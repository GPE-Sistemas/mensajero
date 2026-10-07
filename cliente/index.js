'use strict';

/** status 0: no se llegó al mensajero. 502: Postfix no respondió. En esos dos tiene sentido reintentar. */
class MensajeroError extends Error {
  constructor(status, mensaje) {
    super(mensaje);
    this.name = 'MensajeroError';
    this.status = status;
  }
}

class Mensajero {
  constructor({ url, apikey, timeoutMs = 15000 }) {
    this.url = url.replace(/\/+$/, '');
    this.apikey = apikey;
    this.timeoutMs = timeoutMs;
  }

  async enviar(plantilla, { para, datos, remitente, nombre, responderA }) {
    let res;
    let texto;
    try {
      res = await fetch(`${this.url}/v1/email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${this.apikey}` },
        body: JSON.stringify({ plantilla, para, datos, remitente, nombre, responderA }),
        signal: AbortSignal.timeout(this.timeoutMs),
      });
      texto = await res.text(); // el timeout también cubre la lectura del cuerpo
    } catch (err) {
      throw new MensajeroError(0, `no se pudo llegar al mensajero: ${err.message}${err.cause?.code ? ` (${err.cause.code})` : ''}`);
    }
    let cuerpo;
    try {
      cuerpo = JSON.parse(texto);
    } catch {
      cuerpo = undefined;
    }
    if (!res.ok) {
      const error = cuerpo?.error;
      throw new MensajeroError(res.status, typeof error === 'string' && error ? error : res.statusText || 'error del mensajero');
    }
    if (typeof cuerpo?.id !== 'string') {
      throw new MensajeroError(res.status, 'respuesta inválida del mensajero');
    }
    return { id: cuerpo.id };
  }
}

module.exports = { Mensajero, MensajeroError };
