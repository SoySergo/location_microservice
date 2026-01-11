-- Расширяем поле color для transport_lines с VARCHAR(7) до VARCHAR(50)
-- т.к. в OSM данных могут быть более длинные значения цветов

ALTER TABLE transport_lines 
    ALTER COLUMN color TYPE VARCHAR(50),
    ALTER COLUMN text_color TYPE VARCHAR(50);
