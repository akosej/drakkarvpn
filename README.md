# DrakkarVpn – WireGuard sobre WebSocket
DrakkarVpn es un cliente VPN ligero que combina la potencia de WireGuard con la flexibilidad de WebSockets.
Su arquitectura utiliza un túnel TUN para gestionar las configuraciones de WireGuard y encapsula el tráfico en TCP vía WebSocket, permitiendo que la conexión VPN se disfrace como tráfico HTTPS estándar.

## ✨ Características principales
- 🔒 Seguridad moderna: cifrado de WireGuard con soporte para túneles TUN.

- 🌐 Encapsulación WebSocket: ideal para entornos donde UDP está bloqueado o restringido.

- ⚡ Compatibilidad multiplataforma: pensado para funcionar en Windows, Linux y entornos cloud.

- 🛠️ Configuración sencilla: cliente minimalista con parámetros claros para levantar el túnel.

- 🎭 Evasión de bloqueos: el tráfico se ve como conexiones HTTPS normales, evitando censura y firewalls agresivos.

## 🚀 Casos de uso
- Conexiones seguras en redes con restricciones de UDP.

- Integración con servidores detrás de Nginx/443 para multiplexar tráfico web y VPN.

- Escenarios donde se requiere VPN sobre TCP sin sacrificar rendimiento.