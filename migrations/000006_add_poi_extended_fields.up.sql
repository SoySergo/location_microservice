-- Добавление расширенных полей для POI
-- Контактная информация
ALTER TABLE pois ADD COLUMN email VARCHAR(255);

-- Дополнительная информация
ALTER TABLE pois ADD COLUMN description TEXT;
ALTER TABLE pois ADD COLUMN brand VARCHAR(100);
ALTER TABLE pois ADD COLUMN operator VARCHAR(100);
ALTER TABLE pois ADD COLUMN cuisine VARCHAR(100);
ALTER TABLE pois ADD COLUMN diet VARCHAR(50);

-- Численные характеристики
ALTER TABLE pois ADD COLUMN stars INTEGER CHECK (stars >= 0 AND stars <= 5);
ALTER TABLE pois ADD COLUMN rooms INTEGER CHECK (rooms >= 0);
ALTER TABLE pois ADD COLUMN beds INTEGER CHECK (beds >= 0);
ALTER TABLE pois ADD COLUMN capacity INTEGER CHECK (capacity >= 0);
ALTER TABLE pois ADD COLUMN min_age INTEGER CHECK (min_age >= 0);

-- Услуги и удобства
ALTER TABLE pois ADD COLUMN internet VARCHAR(50);
ALTER TABLE pois ADD COLUMN internet_fee BOOLEAN;
ALTER TABLE pois ADD COLUMN smoking VARCHAR(50);
ALTER TABLE pois ADD COLUMN outdoor_seating BOOLEAN;
ALTER TABLE pois ADD COLUMN takeaway BOOLEAN;
ALTER TABLE pois ADD COLUMN delivery BOOLEAN;
ALTER TABLE pois ADD COLUMN drive_through BOOLEAN;

-- Оплата
ALTER TABLE pois ADD COLUMN fee BOOLEAN;
ALTER TABLE pois ADD COLUMN charge VARCHAR(100);
ALTER TABLE pois ADD COLUMN payment_cash BOOLEAN;
ALTER TABLE pois ADD COLUMN payment_cards BOOLEAN;

-- Социальные сети и контакты
ALTER TABLE pois ADD COLUMN facebook VARCHAR(255);
ALTER TABLE pois ADD COLUMN instagram VARCHAR(255);
ALTER TABLE pois ADD COLUMN twitter VARCHAR(255);

-- Дополнительные поля
ALTER TABLE pois ADD COLUMN image_url TEXT;
ALTER TABLE pois ADD COLUMN wikidata VARCHAR(50);
ALTER TABLE pois ADD COLUMN wikipedia TEXT;

-- Индксы пока не создём
-- CREATE INDEX idx_pois_brand ON pois (brand) WHERE brand IS NOT NULL;
-- CREATE INDEX idx_pois_cuisine ON pois (cuisine) WHERE cuisine IS NOT NULL;
-- CREATE INDEX idx_pois_diet ON pois (diet) WHERE diet IS NOT NULL;
-- CREATE INDEX idx_pois_stars ON pois (stars) WHERE stars IS NOT NULL;
-- CREATE INDEX idx_pois_wheelchair_access ON pois (wheelchair) WHERE wheelchair = true;
-- CREATE INDEX idx_pois_outdoor_seating ON pois (outdoor_seating) WHERE outdoor_seating = true;
-- CREATE INDEX idx_pois_takeaway ON pois (takeaway) WHERE takeaway = true;
-- CREATE INDEX idx_pois_delivery ON pois (delivery) WHERE delivery = true;
