Plan: Search Map Bug Fixes — Iteration 4
TL;DR: 25+ багов распределены между 3 сервисами: realbro_frontend (18 задач), realbro_backend (4 задачи), location (2 задачи). Критичные: ошибка GeoJSON геометрии (бекенд json.Marshal вместо geojson.Encode), несовпадение полей price_per_m2/price_per_meter, потеря данных локаций при смене мода (нет localStorage), восстановление только 1 полигона.

Секция A: Фильтры
A1. Расположение фильтров на ПК (фронтенд)

Файл: src/widgets/search-filters-bar/ui/filters-desktop-panel.tsx
Изменить layout: фильтры прижать к правому краю (ml-auto / justify-end), логотип слева. Возможно затрагивает src/screens/search-map-page/ui/ui.tsx — верхний контейнер.
A2. Категории / субкатегории для UI (бекенд)

Эндпоинты уже существуют: GET /api/v1/dictionaries/categories?lang=X и GET /api/v1/dictionaries/categories/:id/subcategories в internal/delivery/http/v1/dictionary/handler.go.
Проверить, что они возвращают данные в формате, удобном для фронтенда (ID + название). Категории: 1=room, 2=apartment, 3=house.
A3. Фронтенд получает данные по апи (фронтенд)

Добавить API-вызовы в src/shared/api/ для получения категорий/субкатегорий.
В filters-desktop-panel.tsx добавить селекторы property_category (с данными из API) и property_subcategory. property_type = "rent" (дефолт, disabled).
В useFilterStore (store.ts) хранить выбранные categoryIds и subcategoryIds.
A4. Логика фильтра с задержкой (фронтенд)

В src/widgets/search-filters-bar/model/store.ts добавить debounce (300-500мс) при изменении полей фильтров перед отправкой запросов на short-listing и tiles.
A5. Сохранение данных мода локаций в localStorage (фронтенд)

Файл: src/features/location-search-mode/model/hooks/use-search-mode-state.ts — сейчас useState, нет персистенции.
Реализовать: при изменении данных в каждом моде (поиск, рисование) — записывать в localStorage (ключ location-mode-{modeName}). При переключении мода — восстанавливать из localStorage. При закрытии мода локаций — очищать все ключи.
A6. Полигоны городов/районов исчезают после загрузки (фронтенд)

Файлы: src/features/location-search-mode/model/hooks/use-boundaries-hover.ts и связанные хуки useBoundariesLayer.
Проблема в гонке: слои добавляются, но при смене activeLocationMode — источник данных очищается или видимость переключается. Нужно добавить проверку, что слой видим только когда поиск активен, но не убирать данные при toggle.
A7. Ошибка сохранения геометрии — invalid GeoJSON (бекенд)

Корневая причина: internal/repository/filter/repository.go строки 335-345 — функция geometryToGeoJSON() использует json.Marshal(geom) на go-geom.T, что сериализует внутреннюю Go-структуру, а не валидный GeoJSON.
Исправление: использовать geojson.Encode(geomT) из github.com/twpayne/go-geom/encoding/geojson для корректной конвертации. Альтернативный вариант — передавать оригинальную GeoJSON-строку из запроса прямо в SQL, минуя decode→encode roundtrip.
A8. Восстановление только 1 полигона при переоткрытии (фронтенд)

Файл: src/widgets/search-filters-bar/model/store.ts ~строка 180 — convertFiltersToLocationFilter() использует filters.geometryIds[0] — берёт только первый.
Исправление: итерировать по всем filters.geometryIds и восстанавливать все полигоны.
A9. Тестирование фильтров с маркерами + локациями (бекенд)

Параметры polygon_ids, exclude_ids, include_ids, location_ids уже проходят в SQL через toListParams(), toCountParams(), toMVTParams().
Необходимо написать интеграционные тесты с комбинациями: polygon + location + markers.
Секция B: Карточки объекта
B1. Отображение bedrooms/rooms (фронтенд)

Бекенд возвращает bedrooms в PaginatedShortListResponse. Фронтенд не читает это поле.
Добавить bedrooms?: number в api-types.ts (PropertyShortListingDTO).
Добавить bedrooms?: number в card-types.ts (PropertyGridCard).
В converters.ts маппить dto.bedrooms.
В property-card-grid/ui.tsx ~строка 278 — рендерить оба значения если присутствуют: «N комнат · M спален».
B2.1. Фильтрация транспортных линий при обогащении (location сервис)

Файл: internal/repository/postgresosm/transport_repository.go:
Строки 1195-1200: изменить priority_rank с 2-уровневой на 4-уровневую систему: metro=1, train=2, tram=3, bus=4.
Строка 1497 (GetLinesByStationIDsBatch): добавить фильтр WHERE l.ref IS NOT NULL AND l.ref != '' — исключить линии без ref (именно они содержат длинные описательные имена типа "602 Figueres - Girona - Barcelona...").
Обновить логику в transport_usecase.go строка 280 — hasHighPriority теперь должен учитывать 4 ранга.
B2.2. Лимитирование станций + стиль hover + ref вместо name (фронтенд)

Файл: converters.ts строка 53 — заменить line.name на line.ref || line.name при маппинге.
Файл: property-card-grid/ui.tsx:
Строка 345: рендерить line.ref || line.name || 'M'.
Добавить MAX_VISIBLE_LINES = 3 лимит для отрисовки.
Бекграунд при hover — убрать реверс цвета, использовать стиль приложения (тёмная тема = тёмный фон попапа).
B3. Agency slug undefined (фронтенд + бекенд)

Бекенд: internal/domain/company/entity.go — Slug *string может быть nil. DTO с omitempty отбрасывает пустой slug.
Опция 1 (бекенд): при создании/обновлении компании генерировать slug из имени, если он nil.
Опция 2 (фронтенд): property-card-grid/ui.tsx строка 216 — если slug и id пусты, не рендерить ссылку на агентство. href="/agency/${property.author.slug || property.author.id || ''}" — добавить проверку.
B4. price_per_m2 не отрисовывается (фронтенд)

Причина: бекенд возвращает поле price_per_m2 (JSON), а фронтенд ожидает price_per_meter.
Файл: api-types.ts строка 133 — переименовать price_per_meter → price_per_m2.
Файл: converters.ts строка 35 — обновить маппинг.
Файл: card-types.ts — обновить интерфейс.
Файл: property-card-grid/ui.tsx строка 246 — обновить обращение к полю.
Секция C: Список (Сайдбар)
C1. Пагинация по скролу (фронтенд)

Файл: src/screens/search-list-page/ui/ui.tsx ~строка 230 — заменить компонент <Pagination> на IntersectionObserver для infinite scroll.
Использовать подход как в MapSidebar: src/widgets/map-sidebar/ui/ui.tsx ~строка 350 — onScroll + cursor из ответа бекенда.
Бекенд уже поддерживает cursor-based пагинацию (next_cursor, has_more).
C2. Каунт не отображается в сайдбаре (фронтенд)

Бекенд корректно возвращает { "data": { "count": 78 } }.
Проверить в фронтенде путь доступа к данным: возможно читается response.count вместо response.data.count.
Файлы: src/widgets/listing-controls/ui/ui.tsx, src/widgets/map-sidebar/ui/ui.tsx строка 307.
C3. Точки на карте: цена + кластеры + карточка (фронтенд)

Файл: src/features/map/ui/search-map/ui.tsx:
Одиночный объект: заменить слой circle (~строка 125) на symbol с текстом цены.
Кластер: уже показывает point_count_abbreviated — ок.
Клик по одиночному объекту → попап карточка на карте.
Клик по кластеру → в сайдбаре отображаются объекты кластера, кнопка «Закрыть» для возврата к обычному списку.
C4. Hover на карточку → подсветка на карте (фронтенд)

Файл: src/screens/search-map-page/ui/ui.tsx ~строка 101 — handlePropertyHover сейчас только console.log.
Реализовать: при hover добавлять временный маркер (иконка домика в круге) на координаты объекта поверх всех слоёв.
C5. Обновление сайдбара при движении карты (фронтенд)

Файл: src/features/map/ui/search-map/ui.tsx — нет обработчика moveend.
Добавить map.on('moveend', ...) → извлечь bounds → передать как bbox в запрос short-listing для обновления сайдбара.
C6. Синхронизация с фильтрами (фронтенд)

При изменении любого фильтра → перезапрос count, short-listing, tiles. Убедиться, что useFilterStore триггерит все три запроса.
C7. Компактные контролы в сайдбаре (фронтенд)

Файл: src/widgets/listing-controls/ui/ui.tsx.
Редизайн: слева — каунт, справа — 2 иконн-кнопки (список/карта, сортировка) без текста, с tooltip при hover.
C8. Кнопка «Закрыть» для кластера (фронтенд)

Когда в сайдбаре показан кластер — заменить контролы на кнопку «Закрыть» для возврата к основному списку.
Секция D: Ошибки консоли
D1. NaN в children (фронтенд)

Файл: src/app/[locale]/search/properties/list/page.tsx строка 55.
Причина: pagination?.total может быть undefined → NaN при передаче в компонент. Добавить fallback: total ?? 0.
D2. Hydration mismatch — timeAgo (фронтенд)

Файл: src/entities/property/ui/property-card-grid/ui.tsx — useMemo с new Date() различается между SSR и клиентом.
Исправление: обернуть timeAgo в useEffect + useState (клиент-only рендер), или использовать suppressHydrationWarning на контейнере.
D3. Отсутствующие переводы filters.show, filters.resultsFound (фронтенд)

Файл: messages/ru.json — в блоке "filters" (начиная со строки 254) нет ключей show и resultsFound.
Добавить: "show": "Показать" и "resultsFound": "Найдено {count} объектов" в секцию "filters".
Проверить все остальные локали (en, es, fr и т.д.).
D4. BaseMap error {} (фронтенд)

Файл: src/features/map/ui/base-map/ui.tsx строка 100.
Некритичная ошибка, возможно из-за Mapbox credentials или handleMouseLeave в cleanup. Добавить логирование деталей ошибки.
D5. Failed to create geometry (фронтенд + бекенд)

Связано с A7. Фронтенд: src/shared/api/geometries.ts строка 170 — createGeometry() делает JSON.stringify(data.geometry) — возможно двойная сериализация если geometry уже строка. Добавить проверку типа перед stringify.
Verification
Бекенд (A7): После исправления geometryToGeoJSON() — POST /api/v1/filters/guest/geometry с валидным GeoJSON Polygon должен вернуть 200.
Фронтенд (B4): Проверить ответ GET /properties/short-listing, подтвердить что price_per_m2 отображается на карточке.
Фронтенд (A5/A8): Переключить мод локации search→draw→search — данные сохраняются. Нарисовать 3 полигона, закрыть→открыть мод — все 3 восстанавливаются.
Location (B2.1): Запрос EnrichLocationBatch — станции возвращаются с приоритетом metro > train > tram > bus, без длинных линий без ref.
Фронтенд (C1): Скролл до конца списка → подгружаются следующие карточки через cursor.
Фронтенд (C3-C5): На карте одиночный объект = цена, клик = карточка. Движение карты обновляет сайдбар. Hover = иконка дома.
Decisions
A7: Использовать geojson.Encode() вместо прокидывания raw-строки — безопаснее для валидации на уровне Go.
B3: Реализовать оба фикса: бекенд генерирует slug при nil + фронтенд graceful fallback.
B4: Переименовать фронтенд поле в price_per_m2 (выровнять с бекенд API).
C1: IntersectionObserver на sentinel-элемент в конце списка, не onScroll — более производительно.