#!/bin/sh
set -eu

directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ "$(id -u)" = 0 ]; then
  echo 'Запустите установщик от обычного пользователя, без sudo.' >&2
  exit 1
fi
"$directory/freeturn-desktop" --install-desktop
echo 'Настройка системного VPN-helper: подтвердите запрос администратора.'
status=0
if ! "$directory/freeturn-desktop" --configure-vpn-helper; then
  echo 'Клиент установлен, но VPN-helper не настроен. При подключении потребуется пароль. Для повторной настройки запустите установщик снова.' >&2
  status=1
fi
printf 'Добавить ярлык на рабочий стол? [д/N]: '
answer=
IFS= read -r answer || :
case "$answer" in
  д|Д|да|Да|y|Y|yes|Yes)
    "$directory/freeturn-desktop" --create-desktop-shortcut || status=1 ;;
esac
exit "$status"
