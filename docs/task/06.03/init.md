# Общее

## Страница Поиск Карта

#### Фильтры
1. Расположение фильтров на пк версии. Они должны прилепать к правой стороне экрана. Слева логотип. 

2. **Бекенду** Нужно вернуть для ui фронтенда категории, суб категории, тип аренды.  
3. **Фронтенд** Получает данные по айпи.  property_type = rent (default, ui disabled), property_category = (бекенд ждёт ids 1 room, 2 apartment, 3 house), property_subcategory 
4. Логика фильтра. Заполняем поля с задежкой идёт запрос объектов для карты тайл и так же для сайдбара, объекты отрисовываются в сайдбаре и на карте.
**Фильтр локаций**
5. При смене мода, данные не востанавливаются, временно хранить нужно в локальном стораже. Если мы выбрали в режиме поиска 5 городов, сменили мод, на рисование, вернулись обратно, там пусто. Логика должна быть, данные сохраняются у пользователя в локал стораже, если он закрывает мод локаций, то очищается стораж.
6. Поиск + полигоны городов, районов. (При загрузке полигоны появляются а после исчезают,  )
7. Ошибка при сохранение геометрии. Логи ниже Лог_1.
8. Если закрыли мод локации, а до этого нариосвали больше чем 1 полигон в моде рисования или другом, то при переоткрытии мода восстанавливается только 1 полигон. 
9. **Для бекенда**: Протестировать работу фильтров с маркерами + локациями


#### Карточки объекта: 
1. Бекенд возвращает bedrooms или rooms. Фронтенд должен отрисовывать в карточках комнаты эти значения. Либо комнат, либо спален, либ ои то и другое, если указано.
2.1. **Для сервиса локаций**: Нужно фильтровать при обогащении данные. Для станции возвращается приоритет метро, поезда(r2 и тд), после уже если ничего нет трамваим и автобусы. Так же иногда попадают названия линий {
              "id": -19307264,
              "name": "602 Figueres - Girona - Barcelona - Aeroport del Prat",
              "type": "bus"
            }, и они на фронтенде растягиваются на всю карточку, должны быть только короткие. ref
2.2. На фронтенде нужно лимитировать отрисовку станций. Сделать бекграунд при ховере в стиле приложения. Не делать реверс цвета. Если тема тёмная, окно тёмное. Фронтенд должен отрисовывать ref а не name

3. Agency slug undefined, разобраться почему и как исправить
4. На фронтенде не отрисовывается цена за кв.м. бекенд возвращает "price_per_m2": 42.97,

#### Список:
1. Пагинация (должно быть по скролу подгрузка данных, не используем клик вперёд, назад)
2. Не высвечивается на фронтенде каунт в сайдбаре. Бекенд возвращает {
  "data": {
    "count": 78
  }
}
3. Точки на карте отображаются не полноценно: Должно быть Единичный объект на карте = цена. Кластер = кол-во объектов. При клике на едииничный объект, на карте появляется карточка. При клике на кластер в сайдбаре появляется кластер, можно закрыть кластер и вернётся к обычному списку, который был до этого. 
4. При ховере на карточку объекта в сайдбаре на карте подсвечивается местоположение объекта иконка домик не в зависимости от того где находится объект в кластере или одиночный, поверх всего появляется дом в круге.
5. При передвижении карты, нужно делать запрос на получение новых объектов для сайдбара с новым квадратом. 
6. Синхронизация с фильтрами 
7. Переключение на "Список" и "сортировка" занимают слишком много места в сайдбаре, нужно сделать компактней. Слева Каунт, справа 2 кнопки без текста, при ховере подсказака появляется
8. Если кластер, то кнопка закрыть вместо этих кнопок.
9. 


# Лог_1 
2026-03-06T17:22:19.356+0100    DEBUG   app/http.go:107 incoming request      {"request_id": "248e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "OPTIONS", "path": "/api/v1/filters/guest/geometry", "ip": "127.0.0.1", "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:148.0) Gecko/20100101 Firefox/148.0"}
2026-03-06T17:22:19.356+0100    DEBUG   app/http.go:138 request completed     {"request_id": "248e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "OPTIONS", "path": "/api/v1/filters/guest/geometry", "status": 204, "latency": 0.00072823}
2026-03-06T17:22:19.358+0100    DEBUG   app/http.go:107 incoming request      {"request_id": "258e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "POST", "path": "/api/v1/filters/guest/geometry", "ip": "127.0.0.1", "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:148.0) Gecko/20100101 Firefox/148.0"}
2026-03-06T17:22:19.383+0100    ERROR   filter/repository.go:251        failed to create filter geometry       {"error": "ERROR: invalid GeoJSON representation (SQLSTATE XX000)", "filterID": "00000000-0000-0000-0000-000000000000"}
2026-03-06T17:22:19.386+0100    ERROR   filter/service.go:311   failed to create guest geometry        {"error": "failed to create filter geometry: ERROR: invalid GeoJSON representation (SQLSTATE XX000)"}
2026-03-06T17:22:19.386+0100    ERROR   filter/handler.go:813   Failed to create guest geometry        {"error": "failed to create guest geometry: failed to create filter geometry: ERROR: invalid GeoJSON representation (SQLSTATE XX000)"}
2026-03-06T17:22:19.387+0100    ERROR   app/http.go:121 request completed     {"request_id": "258e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "POST", "path": "/api/v1/filters/guest/geometry", "status": 500, "latency": 0.028332114}
2026-03-06T17:22:19.404+0100    DEBUG   app/http.go:107 incoming request      {"request_id": "268e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "POST", "path": "/api/v1/filters/guest/geometry", "ip": "127.0.0.1", "user_agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:148.0) Gecko/20100101 Firefox/148.0"}
2026-03-06T17:22:19.407+0100    ERROR   filter/repository.go:251        failed to create filter geometry       {"error": "ERROR: invalid GeoJSON representation (SQLSTATE XX000)", "filterID": "00000000-0000-0000-0000-000000000000"}
2026-03-06T17:22:19.407+0100    ERROR   filter/service.go:311   failed to create guest geometry        {"error": "failed to create filter geometry: ERROR: invalid GeoJSON representation (SQLSTATE XX000)"}
2026-03-06T17:22:19.407+0100    ERROR   filter/handler.go:813   Failed to create guest geometry        {"error": "failed to create guest geometry: failed to create filter geometry: ERROR: invalid GeoJSON representation (SQLSTATE XX000)"}
2026-03-06T17:22:19.407+0100    ERROR   app/http.go:121 request completed     {"request_id": "268e8c5e-d5c7-447e-bdeb-6f06010db962", "method": "POST", "path": "/api/v1/filters/guest/geometry", "status": 500, "latency": 0.002992836}


# Логи ошибок фронтенд:
1. 1/6
Next.js 16.0.1 (stale)Turbopack
Console Error
Received NaN for the `children` attribute. If this is expected, cast the value to a string.

src/app/[locale]/search/properties/list/page.tsx (55:13) @ Page

  53 |     return (
  54 |         <Suspense fallback={<SearchListSkeleton />}>
> 55 |             <SearchListPage
     |             ^
  56 |                 initialData={initialData}
  57 |                 initialFilters={filters}
  58 |                 initialPage={page}

Call Stack 24
Show 22 ignore-listed frame(s)
button
unknown (0:0)
Page
src/app/[locale]/search/properties/list/page.tsx (55:13)
1
2

2. 2/6
Next.js 16.0.1 (stale)Turbopack
Recoverable Error
Hydration failed because the server rendered text didn't match the client. As a result this tree will be regenerated on the client. This can happen if a SSR-ed Client Component used:

- A server/client branch `if (typeof window !== 'undefined')`.
- Variable input such as `Date.now()` or `Math.random()` which changes each time it's called.
- Date formatting in a user's locale which doesn't match the server.
- External changing data without sending a snapshot of it along with the HTML.
- Invalid HTML tag nesting.

It can also happen if the client has a browser extension installed which messes with the HTML before React loaded.

See more info here: https://nextjs.org/docs/messages/react-hydration-error

  ...
    <SearchListPage initialData={{data:[...], ...}} initialFilters={{}} initialPage={1} initialSortBy="createdAt" ...>
      <div className="flex min-h...">
        <HeaderSlot>
        <main className="flex-1 md:...">
          <div>
          <div>
          <ListingControls>
          <MapPreview>
          <AiAgentStories>
          <div className="flex-1 p-3...">
            <div className="grid grid-...">
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid>
              <PropertyCardGrid property={{id:"069a46...", ...}} onClick={function onClick} ...>
                <div className="bg-card ro..." onClick={function onClick} onMouseEnter={function onMouseEnter} ...>
                  <div className="relative a..." onMouseMove={function PropertyCardGrid.useCallback[handleMouseMove]} ...>
                    <img>
                    <div className="absolute t...">
                      <Link href="/agency/un..." className="flex items..." onClick={function onClick}>
                        <BaseLink ref={null} href="/fr/agency..." locale={undefined} localeCookie={{name:"NEXT...", ...}} ...>
                          <LinkComponent ref={null} href="/fr/agency..." hrefLang={undefined} ...>
                            <a hrefLang={undefined} className="flex items..." ref={function} onClick={function onClick} ...>
                              <_c>
                              <span className="text-[10px] text-text-secondary">
+                               il y a 54 min
-                               il y a 53 min
                  ...
              ...
          ...
        ...

src/screens/search-list-page/ui/ui.tsx (199:33) @ SearchListPage/<.children<.children<.children<.children<

  197 |                         <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-2 sm:gap-3 md:gap-4">
  198 |                             {properties.map((property) => (
> 199 |                                 <PropertyCardGrid
      |                                 ^
  200 |                                     key={property.id}
  201 |                                     property={property}
  202 |                                     onClick={() => handlePropertyClick(property)}

Call Stack 21
Show 17 ignore-listed frame(s)
span
unknown (0:0)
SearchListPage/<.children<.children<.children<.children<
src/screens/search-list-page/ui/ui.tsx (199:33)
SearchListPage
src/screens/search-list-page/ui/ui.tsx (198:41)
Page
src/app/[locale]/search/properties/list/page.tsx (55:13)

3. 3/6
Next.js 16.0.1 (stale)Turbopack
Console Error
MISSING_MESSAGE: Could not resolve `filters.show` in messages for locale `ru`.

src/widgets/search-filters-bar/ui/filters-desktop-panel.tsx (513:46) @ FiltersDesktopPanel

  511 |                                         <>
  512 |                                             <SlidersHorizontal className="w-4 h-4 mr-2" />
> 513 |                                             {t('show')}
      |                                              ^
  514 |                                         </>
  515 |                                     )}
  516 |                                 </Button>

Call Stack 23
Show 18 ignore-listed frame(s)
FiltersDesktopPanel
src/widgets/search-filters-bar/ui/filters-desktop-panel.tsx (513:46)
SearchFiltersBarContent
src/widgets/search-filters-bar/ui/ui.tsx (431:13)
SearchFiltersBar
src/widgets/search-filters-bar/ui/ui.tsx (455:13)
SearchMapPage
src/screens/search-map-page/ui/ui.tsx (123:17)
Page
src/app/[locale]/search/properties/map/page.tsx (16:13)
1
2

4. 4/6
Next.js 16.0.1 (stale)Turbopack
Console Error
MISSING_MESSAGE: Could not resolve `filters.resultsFound` in messages for locale `ru`.

src/widgets/search-filters-bar/ui/filters-desktop-panel.tsx (289:42) @ FiltersDesktopPanel

  287 |                                     <div className="w-1.5 h-1.5 rounded-full bg-brand-primary animate-pulse" />
  288 |                                     <span>
> 289 |                                         {t('resultsFound', { count: formatNumber(resultsCount) })}
      |                                          ^
  290 |                                     </span>
  291 |                                 </div>
  292 |                             </div>

Call Stack 21
Show 16 ignore-listed frame(s)
FiltersDesktopPanel
src/widgets/search-filters-bar/ui/filters-desktop-panel.tsx (289:42)
SearchFiltersBarContent
src/widgets/search-filters-bar/ui/ui.tsx (431:13)
SearchFiltersBar
src/widgets/search-filters-bar/ui/ui.tsx (455:13)
SearchMapPage
src/screens/search-map-page/ui/ui.tsx (123:17)
Page
src/app/[locale]/search/properties/map/page.tsx (16:13)

5. Console Error
[BaseMap] Map error: {}

src/features/map/ui/base-map/ui.tsx (100:21) @ BaseMap.useEffect

   98 |
   99 |         map.current.on('error', (e) => {
> 100 |             console.error('[BaseMap] Map error:', e);
      |                     ^
  101 |         });
  102 |
  103 |         // Cleanup при размонтировании

Call Stack 14
Show 12 ignore-listed frame(s)
BaseMap.useEffect
src/features/map/ui/base-map/ui.tsx (100:21)
useBoundariesHover.useEffect.handleMouseLeave
src/features/location-search-mode/model/hooks/use-boundaries-hover.ts (84:25)

6. 6/6
Next.js 16.0.1 (stale)Turbopack
Console ApiError
Failed to create geometry

src/shared/api/lib/api-client.ts (27:9) @ ApiError

  25 |         public response?: ErrorResponse
  26 |     ) {
> 27 |         super(message);
     |         ^
  28 |         this.name = 'ApiError';
  29 |     }
  30 | }

Call Stack 2
ApiError
src/shared/api/lib/api-client.ts (27:9)
request
src/shared/api/lib/api-client.ts (184:15)
