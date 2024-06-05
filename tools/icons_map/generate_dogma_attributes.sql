-- List of EveDogmaAttribute to define constants
SELECT CONCAT("EveDogmaAttribute", REPLACE(x.attributeName, " ", ""), " = ", x.attributeID )
FROM eve_sde.dgmAttributeTypes x
WHERE categoryID = 1
AND published  IS TRUE
AND iconID  is not NULL;

-- List of EveDogmaAttribute to be included in another package
SELECT CONCAT("model.EveDogmaAttribute", REPLACE(x.attributeName, " ", ""), "," )
FROM eve_sde.dgmAttributeTypes x
WHERE categoryID = 1
AND published  IS TRUE
AND iconID  is not NULL;