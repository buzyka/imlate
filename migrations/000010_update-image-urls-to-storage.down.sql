
UPDATE visitors
SET image = REPLACE(image, '/storage/img/visitors/', '/assets/img/visitors/')
WHERE image LIKE '/storage/img/visitors/%';
