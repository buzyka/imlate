
UPDATE visitors
SET image = ''
WHERE image LIKE '/assets/img/students/%';

UPDATE visitors
SET image = REPLACE(image, '/assets/img/visitors/', '/storage/img/visitors/')
WHERE image LIKE '/assets/img/visitors/%';
