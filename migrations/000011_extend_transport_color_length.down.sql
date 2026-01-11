-- Откат расширения color полей - возвращаем к VARCHAR(7)

ALTER TABLE transport_lines 
    ALTER COLUMN color TYPE VARCHAR(7),
    ALTER COLUMN text_color TYPE VARCHAR(7);
