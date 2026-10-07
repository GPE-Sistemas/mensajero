import { Mensajero } from './index';

const m = new Mensajero<'gas'>({ url: 'http://x', apikey: 'k' });

export async function prueba(): Promise<void> {
  await m.enviar('reset-password', { para: 'a@x.com', datos: { link: 'l' } });
  await m.enviar('scada-reestablecido', { para: 'a@x.com', remitente: 'alertas', nombre: 'Camuzzi', datos: { fecha: '', punto: '', valor: '', variable: '' } });
  // @ts-expect-error plantilla inexistente
  await m.enviar('no-existe', { para: 'a@x.com', datos: {} });
  // @ts-expect-error falta link
  await m.enviar('reset-password', { para: 'a@x.com', datos: {} });
  // @ts-expect-error sobra un campo
  await m.enviar('reset-password', { para: 'a@x.com', datos: { link: 'l', token: 't' } });
  // @ts-expect-error remitente que gas no tiene
  await m.enviar('reset-password', { para: 'a@x.com', remitente: 'otro', datos: { link: 'l' } });
}
