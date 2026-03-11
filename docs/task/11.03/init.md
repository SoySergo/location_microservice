# Панель фильтров
1.1. Недвижимость -> Комната -> Убрать из фильтров субкатегорию.
1.2. Адаптивность (на версии 1366px если заполняем фильтры они залазят на логотип. Этого нужно избежать. Сократить кнопки, задать фиксированную ширину. Убарть из Цена 1200€-2500€ слово цена)
1.3. Синхронизация в во время изменения фильтра у нас в панеле появляется состояние облака с шестерёнокой и тд. Должно быть следующее:   
    **Изменился фильтр** 
    - иконка <CloudCog /> с шестерёнкой
    - запрос на бекенд для получения каунта и списка объектов, так же если карта, то тоже параметры запроса полигона обновляем
    - и после изменения фильтра, можем где-то через пару секунд отправить на апдейт фильтра в бекенд. Пайплайн ui: 
        как делаем запрос на апдейт <CloudSync /> и крутим стрелки. После либо алерт <CloudAlert /> если ошибка, либо <CloudCheck /> если всё ок с зелёным чекером, после чего меняется на неактивный (если изменений нет)
    - в целом логика обновления фильтра уже близка к правильной, но есть проблемы с запросом. Если меняю цену, площадь или количество комнат, не происходит обновления объектов на пк версии
1.4. Переработать детали фильтров, подогнать под дизайн из примера

        


2. Мод Локация: 
2.1. При загрузке в режиме поиск. Идёт запрос для получения административных границ. Изначально они отрисовываются. После исчезают (через пол секунды где-то) пофиксить это




Console Error
[BaseMap] Map error: "The source 'boundaries' does not exist in the map's style." {}

src/features/map/ui/base-map/ui.tsx (108:25) @ BaseMap.useEffect

  106 |             const errorMessage = e.error?.message;
  107 |             if (errorMessage) {
> 108 |                 console.error('[BaseMap] Map error:', errorMessage, e);
      |                         ^
  109 |             }
  110 |         });
  111 |

Call Stack 129
Show 126 ignore-listed frame(s)
BaseMap.useEffect
src/features/map/ui/base-map/ui.tsx (108:25)
useBoundariesHover.useEffect.handleMouseLeave
src/features/location-search-mode/model/hooks/use-boundaries-hover.ts (84:25)
BaseMap.useEffect
src/features/map/ui/base-map/ui.tsx (88:23)

2._. При сохранение полигона:
Console ApiError
Internal server error

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

**backend**
2026-03-11T09:57:44.382+0100    ERROR   middlewares/error_handler.go:43     Unhandled error {"path": "/api/v1/filters/query_1772834890393_e4najj0li/geometry", "method": "POST", "requestID": "21c12593-1390-4aa0-9e53-cbb15ff5beff", "userID": "d55f08f9-663e-4cb3-85ca-2ff46a94dbd2", "error": "Invalid filter ID format"}


**frontend**

Error cleaning up map layers: TypeError: can't access property "getOwnLayer", this.style is undefined
    getLayer map.ts:3320
    removeLayerIfExists map-layer-helpers.ts:7
    cleanupDrawingLayers map-layer-helpers.ts:27
    useEffect ui.tsx:243
    react_stack_bottom_frame react-dom-client.development.js:28022
    runWithFiberInDEV react-dom-client.development.js:987
    commitHookEffectListUnmount react-dom-client.development.js:13678
    commitHookPassiveUnmountEffects react-dom-client.development.js:13709
intercept-console-error.ts:42:26
Uncaught TypeError: can't access property "style", map.getCanvas() is undefined
    useEffect ui.tsx:255
    react_stack_bottom_frame react-dom-client.development.js:28022
    runWithFiberInDEV react-dom-client.development.js:987
    commitHookEffectListUnmount react-dom-client.development.js:13678
    commitHookPassiveUnmountEffects react-dom-client.development.js:13709
    commitPassiveUnmountEffectsInsideOfDeletedTree_begin react-dom-client.development.js:17486
    recursivelyTraversePassiveUnmountEffects react-dom-client.development.js:17307

# Поиск листинг

# Проработать хедер и навигацию, сайдбар

# Чат и общение с ии
1. Перегнать стили из шаблона ai studio


# Детали объекта
1. Хедер, мешает лого, убрать лого вообще с хедера тут.
2. Не отображается ближайший транспорт вверху страницы.
3. При открытии фото на весь экран, нужно что бы открывались поверх сайдбара
4. Нет брекпоинтов для заметок на бекенде
5. Отображение атрибутов (
    "building_info": [],
    "estate_info": [],
    "energy_efficiency": [],
    "characteristics": [],
    "amenities": [],
    "tenant_preferences": [],
    "tenants": [],
        )
    во первых бекенд н евозвращает переводы по локалиям в объекте, во вторых нужно синхронизировать иконки и сделать на бекенде логику. Когда мы обрабатываем объекты при сохранении и добавляем атрибуты создаём и тд, то присваиваем ключ имя для иконок и синхронизации с фронтендом. Фронтенд по ключу выбирает подходящие иконки.
6. Удобства и другие блоки не должны скрываться открываться (если много элементов)
7. Нет отображения ближайших станций метро над картой
8. Карта ниже не отображает POI интересов, транспорт и др. Нужно что бы запрашивались pbf с сервиса локаций а так же сайдбар под картой запрашивал с локацию объекты соответсвующие категории
9. Карточка собственника/агента внизу. Не учитывается агент/владелец. Сейчас данные "author": {
      "contact_id": "069a46fd-0345-7029-8000-ea79ccef331b",
      "name": "Chehsikamehta",
      "author_type": "owner",
      "is_verified": true,
      "object_count": 0,
      "rating": 0,
      "review_count": 0
    }, и отображает "Риелтор". Так же нужно дополнительно возвращать рейтинг, отзывы и внедрять в страницу
10. Ниже объектов должно быть список похожих объектов, если нет других объектов у автора. Если есть то сначала список других объектов автора
11. Сайдбар правый на пк. (Ломается вёрстка на пк версии (![alt text](image.png))) Так же если нет истории изменения цены не нужно кнопку эту отображать
12. Фин условия в сайдбаре:  "deposit_months": 2,
    "deposit": 1200, а на странице Залог —
13. Не корректно так же отображается "объектов в работе" = 0, хотя как минимум 1 если мы на этой странице
14. Не работает логика с описанием, мы достаём в базе перевод + оригинал, возвращаем на фронтенд, если на фронтенде понимаем, что нет перевода, то делаем перевод и подложку показываем оригинал и сверху, что переводится, после меняем на перевод и отправляем на бекенд перевод. Связываем с объектом. Перевод черезе google translate



**backend + scrap**
1. Не обрабатываются корректно данные. look_tenants и другие по slug возвращается: 
    "building_info": [],
    "estate_info": [],
    "energy_efficiency": [],
    "characteristics": [
      {
        "label": "3 habitaciones",
        "value": "3_habitaciones",
        "icon_type": ""
      },
      {
        "label": "1 baño",
        "value": "1_bano",
        "icon_type": ""
      },
      {
        "label": "Terraza",
        "value": "terraza",
        "icon_type": ""
      },
      {
        "label": "Alguien entre 18 y 99 años",
        "value": "alguien_entre_18_y_99_anos",
        "icon_type": ""
      },
      {
        "label": "El género da igual",
        "value": "el_genero_da_igual",
        "icon_type": ""
      },
      {
        "label": "Estancia mínima de 1 mes",
        "value": "estancia_minima_de_1_mes",
        "icon_type": ""
      },
      {
        "label": "Ya disponible",
        "value": "ya_disponible",
        "icon_type": ""
      },
      {
        "label": "Habitación en chalet de 80 m²",
        "value": "habitacion_en_chalet_de_80_m2",
        "icon_type": ""
      },
      {
        "label": "Planta 1ª exterior con ascensor",
        "value": "planta_1a_exterior_con_ascensor",
        "icon_type": ""
      },
      {
        "label": "Capacidad máxima 3 personas",
        "value": "capacidad_maxima_3_personas",
        "icon_type": ""
      },
      {
        "label": "Conexión a internet",
        "value": "conexion_a_internet",
        "icon_type": ""
      },
      {
        "label": "Cocina equipada: nevera, horno, microondas",
        "value": "cocina_equipada_nevera_horno_microondas",
        "icon_type": ""
      }
    ],
    "amenities": [
      {
        "label": "Armarios empotrados",
        "value": "armarios_empotrados",
        "icon_type": ""
      },
      {
        "label": "Cocina equipada",
        "value": "cocina_equipada",
        "icon_type": ""
      },
      {
        "label": "Lavadora",
        "value": "lavadora",
        "icon_type": ""
      },
      {
        "label": "Horno",
        "value": "horno",
        "icon_type": ""
      },
      {
        "label": "Microondas",
        "value": "microondas",
        "icon_type": ""
      },
      {
        "label": "Frigorífico",
        "value": "frigorifico",
        "icon_type": ""
      },
      {
        "label": "Internet / WiFi",
        "value": "internet_wifi",
        "icon_type": ""
      },
      {
        "label": "Jardín",
        "value": "jardin",
        "icon_type": ""
      },
      {
        "label": "Tamaño de la habitación: 30 m²",
        "value": "tamano_de_la_habitacion_30_m2",
        "icon_type": ""
      },
      {
        "label": "Habitación amueblada",
        "value": "habitacion_amueblada",
        "icon_type": ""
      },
      {
        "label": "Armario empotrado",
        "value": "armario_empotrado",
        "icon_type": ""
      },
      {
        "label": "Cama doble",
        "value": "cama_doble",
        "icon_type": ""
      },
      {
        "label": "Ventana a la calle",
        "value": "ventana_a_la_calle",
        "icon_type": ""
      },
      {
        "label": "Chicos y chicas",
        "value": "chicos_y_chicas",
        "icon_type": ""
      },
      {
        "label": "Entre 1 y 99 años",
        "value": "entre_1_y_99_anos",
        "icon_type": ""
      },
      {
        "label": "Trabajan",
        "value": "trabajan",
        "icon_type": ""
      },
      {
        "label": "El propietario/a vive en la casa",
        "value": "el_propietario_a_vive_en_la_casa",
        "icon_type": ""
      },
      {
        "label": "Ambiente de la casa: amistoso, suelen tener visitas",
        "value": "ambiente_de_la_casa_amistoso_suelen_tener_visitas",
        "icon_type": ""
      }
    ],
    "tenant_preferences": [],
    "tenants": [],

