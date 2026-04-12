# Task: Customisation V1 для tracking page `/`

## Кратко
Нужно добавить возможность для admin users настраивать branding tracking page `/` через administration UI.

В v1 в scope входят:
- `favicon`
- `logo_background`
- `welcome_animation`
- `goodbye_animation`
- `welcome_duration_ms`
- `goodbye_duration_ms`

Кастомизация применяется глобально на весь инстанс. Multi-tenant/per-terminal branding в v1 не поддерживается.

## Цель
Дать администратору возможность без деплоя и ручной замены файлов:
- загрузить собственные изображения для tracking page
- сбросить кастомный asset к дефолтному
- настроить время показа welcome/goodbye анимаций
- сразу видеть preview в admin UI

## Текущее состояние
Сейчас tracking page `/` отдается из `website/reader.html` как статический файл.

Внутри страницы захардкожены:
- `assets/img/favicon.ico`
- `assets/img/default-logo.png`
- `assets/img/welcome-images-server.gif`
- `assets/img/good-bye.gif`
- timeout `1800` ms для скрытия анимации и возврата в исходное состояние

В админке отдельного раздела theme/settings нет.

На backend уже есть пример multipart upload для visitor image:
- `POST /admin-api/visitors/:id/image`

На frontend уже есть пример file upload в admin UI:
- `isb-front/src/views/Users.vue`

## Решение v1
### 1. Theme storage
Добавить файловое хранилище темы:
- директория: `/storage/theme`
- source of truth для метаданных: `/storage/theme/theme.json`

В `theme.json` хранить:
- назначение файлов по слотам
- метаданные файлов
- значения duration-настроек

### 2. Theme slots
Поддержать 4 слота:
- `favicon`
- `logo_background`
- `welcome_animation`
- `goodbye_animation`

Правила форматов:
- `favicon`: только PNG
- `logo_background`: JPEG/PNG
- `welcome_animation`: только GIF
- `goodbye_animation`: только GIF

### 3. Theme settings
Добавить 2 настройки:
- `welcome_duration_ms`
- `goodbye_duration_ms`

Хранить в миллисекундах.
Frontend может показывать секунды, но API и manifest должны использовать миллисекунды.

### 4. Public page integration
`GET /` больше не должен просто отдавать статический `reader.html`.

Нужно:
- рендерить `reader.html` как шаблон
- подставлять текущие asset URLs
- подставлять `welcome_duration_ms` и `goodbye_duration_ms`
- использовать дефолтные значения, если кастомизация не задана

Fallback URLs:
- `/assets/img/favicon.ico`
- `/assets/img/default-logo.png`
- `/assets/img/welcome-images-server.gif`
- `/assets/img/good-bye.gif`

### 5. Admin UI structure
В `isb-front` добавить пункт меню `Settings`.

Структура:
- route: `/settings`
- внутри страницы `Settings` использовать tabs
- первая вкладка: `Theme`

Отдельный route `/theme` не делать.

### 6. Universal image helper
Сделать отдельный универсальный helper для обработки изображений.

Требования:
- использовать его в theme upload
- спроектировать так, чтобы позже можно было переиспользовать для visitor photo upload
- поддержать MIME sniffing
- валидировать размер файла
- валидировать допустимые форматы
- для JPEG/PNG уметь уменьшать изображение до web-friendly размера
- использовать безопасные имена файлов
- использовать versioned filenames для cache busting

Поведение для GIF:
- не конвертировать GIF в другой формат
- не ломать анимацию
- если GIF слишком большой и не проходит ограничения, отклонять upload понятной ошибкой

## Backend changes
### Конфиг
Добавить в config:
- `THEME_DIR`, default `storage/theme`
- `THEME_URL_PREFIX`, default `/storage/theme`

### Admin API
Добавить endpoints:
- `GET /admin-api/theme`
- `POST /admin-api/theme/assets/:slot`
- `DELETE /admin-api/theme/assets/:slot`
- `PUT /admin-api/theme/settings`

`GET /admin-api/theme` должен возвращать:
- текущие данные по слотам
- дефолтные URLs
- флаг `is_custom`
- content type
- size bytes
- updated_at
- settings с `welcome_duration_ms` и `goodbye_duration_ms`

### Reader template
Нужно убрать хардкоды из `website/reader.html`:
- favicon path
- logo background path
- welcome gif path
- goodbye gif path
- timeout `1800`

Все эти значения должны приходить из theme config.

### SPA routing
Backend должен раздавать admin SPA entrypoint для:
- `/admin`
- `/admin/users`
- `/admin/admin-users`
- `/admin/login`
- `/admin/settings`

## Frontend changes
В `isb-front` нужно:
- добавить пункт меню `Settings`
- добавить route `/settings`
- добавить `SettingsView`
- сделать tabs внутри `SettingsView`
- первая вкладка `Theme`

Во вкладке `Theme` должны быть:
- карточки preview для 4 слотов
- upload action для каждого слота
- reset action для каждого слота
- поля для `welcome_duration_ms` и `goodbye_duration_ms`
- сохранение duration settings
- загрузка текущего состояния с backend
- обработка ошибок upload/settings save
- подсказки по допустимым форматам и ограничениям

Для dev-режима нужен proxy:
- `/storage` -> backend

## Предлагаемый contract theme.json
Структура может быть примерно такой:

```json
{
  "assets": {
    "favicon": {
      "file_name": "favicon-1712345678.png",
      "url": "/storage/theme/favicon-1712345678.png",
      "content_type": "image/png",
      "size_bytes": 12345,
      "is_custom": true,
      "updated_at": "2026-03-31T12:00:00Z"
    },
    "logo_background": {
      "file_name": "logo_background-1712345678.jpg",
      "url": "/storage/theme/logo_background-1712345678.jpg",
      "content_type": "image/jpeg",
      "size_bytes": 45678,
      "is_custom": true,
      "updated_at": "2026-03-31T12:00:00Z"
    },
    "welcome_animation": {
      "file_name": "welcome_animation-1712345678.gif",
      "url": "/storage/theme/welcome_animation-1712345678.gif",
      "content_type": "image/gif",
      "size_bytes": 99999,
      "is_custom": true,
      "updated_at": "2026-03-31T12:00:00Z"
    },
    "goodbye_animation": {
      "file_name": "goodbye_animation-1712345678.gif",
      "url": "/storage/theme/goodbye_animation-1712345678.gif",
      "content_type": "image/gif",
      "size_bytes": 99999,
      "is_custom": true,
      "updated_at": "2026-03-31T12:00:00Z"
    }
  },
  "settings": {
    "welcome_duration_ms": 1800,
    "goodbye_duration_ms": 1800
  }
}
```

Это пример структуры. Итоговая реализация должна сохранить те же semantics:
- assets
- settings
- slot-based mapping
- file metadata
- explicit duration values

## Разделение задачи между агентами
### Агент 1: Backend admin-api и theme manifest
Зона ответственности:
- theme manifest
- чтение/запись `theme.json`
- usecase layer для theme
- endpoints:
  - `GET /admin-api/theme`
  - `POST /admin-api/theme/assets/:slot`
  - `DELETE /admin-api/theme/assets/:slot`
  - `PUT /admin-api/theme/settings`
- валидация slot names
- удаление старого custom file после успешной замены asset
- возврат fallback/default state если manifest отсутствует

Результат:
- готовый backend API для управления темой
- unit tests и handler tests

### Агент 2: Universal image helper
Зона ответственности:
- отдельный reusable helper для image processing
- MIME sniffing
- ограничения по типам и размеру
- уменьшение JPEG/PNG
- сохранение GIF без поломки анимации
- генерация безопасных versioned filenames
- возвращаемые metadata для manifest/API

Результат:
- независимый helper, который можно позже переиспользовать для visitor photo upload
- unit tests на image processing

### Агент 3: Backend public page integration
Зона ответственности:
- перевод `GET /` на template rendering
- подстановка theme asset URLs
- подстановка `welcome_duration_ms` и `goodbye_duration_ms`
- fallback на дефолтные assets/settings
- admin SPA routing для `/admin/settings`

Результат:
- tracking page реально использует настройки темы
- админка открывается по `/admin/settings`

### Агент 4: Frontend admin UI
Зона ответственности:
- пункт меню `Settings`
- route `/settings`
- `SettingsView` с tabs
- вкладка `Theme`
- preview/upload/reset для 4 слотов
- редактирование 2 duration settings
- интеграция с backend API
- обработка loading/error/success states
- proxy `/storage` в dev

Результат:
- админ может полностью управлять темой из UI

## Acceptance criteria
- Admin может открыть `Settings -> Theme`
- Admin видит текущие значения всех 4 слотов и 2 duration settings
- Admin может загрузить custom asset в любой поддерживаемый слот
- После upload preview обновляется и использует новый URL
- Tracking page `/` начинает использовать новый asset без ручного редактирования файлов
- Admin может сбросить любой слот к дефолтному asset
- Admin может задать разное время показа для welcome и goodbye GIF
- Tracking page использует именно заданные durations
- При отсутствии theme manifest система корректно работает на дефолтных файлах
- При недопустимом формате или слишком большом файле admin получает понятную ошибку
- GIF остаются анимированными и не конвертируются

## Test plan
### Backend
- `GET /admin-api/theme` без manifest
- upload валидного PNG в `favicon`
- upload валидного JPEG/PNG в `logo_background`
- upload валидного GIF в `welcome_animation`
- upload валидного GIF в `goodbye_animation`
- upload в несуществующий slot
- upload файла неподдерживаемого MIME
- upload слишком большого файла
- поврежденный файл
- ошибка записи на диск
- `PUT /admin-api/theme/settings` с валидными duration values
- `PUT /admin-api/theme/settings` с нулевыми/отрицательными значениями
- `DELETE /admin-api/theme/assets/:slot`
- `GET /` с manifest
- `GET /` без manifest

### Frontend
- открытие `/settings`
- отображение вкладки `Theme`
- загрузка current theme state
- preview дефолтного asset
- preview кастомного asset
- upload success
- upload error
- reset success
- save settings success
- save settings error
- корректная работа в dev с `/storage` proxy

## Риски и замечания
- `reader.html` сейчас статический, поэтому потребуется аккуратно перевести его на шаблон, не сломав текущую JS-логику tracking.
- Нельзя полагаться на текущее захардкоженное значение `1800` как на единственную бизнес-константу; duration должны стать явной частью theme settings.
- При замене asset нужно учитывать browser cache, поэтому filenames должны быть versioned.
- Если later helper будет переиспользован для visitor photos, важно не завязать его только на theme-specific slot rules.

## Короткие issue для GitHub
### Backend
**Title:** Настройки темы tracking page и upload API

**Description:**  
Добавить глобальную theme-конфигурацию для `/`, хранение файлов в `/storage/theme`, manifest `theme.json`, admin-api endpoints для получения/upload/reset ассетов и сохранения duration-настроек welcome/goodbye, шаблонный рендер `reader.html` и универсальный image helper для статических изображений.

### Frontend
**Title:** Settings > Theme в админке

**Description:**  
Добавить в `isb-front` страницу `/settings` с вкладкой `Theme`, preview и загрузкой 4 branding-ассетов tracking page, полями длительности welcome/goodbye, интеграцией с `/admin-api/theme` endpoints, reset до дефолта и proxy `/storage` для локальной разработки.

## Принятые допущения
- v1 ограничен 4 image slots и 2 duration settings
- тексты, цвета и layout tracking page не меняются
- тема одна на весь инстанс
- метаданные темы хранятся в файле, без новой таблицы БД
- GIF не конвертируются
- `Theme` живет внутри `Settings` как первая вкладка
