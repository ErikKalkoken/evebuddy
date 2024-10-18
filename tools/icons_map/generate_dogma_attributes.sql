SELECT
	CONCAT("EveDogmaAttribute", REPLACE(dat.attributeName, " ", ""), " = ", dat.attributeID ) as model,
	CONCAT("model.EveDogmaAttribute", REPLACE(dat.attributeName, " ", ""), "," ) as reference
FROM eve_sde.dgmAttributeTypes dat
JOIN dgmTypeAttributes dta ON dta.attributeID = dat.attributeID
WHERE dta.typeID = 22119
AND published  IS TRUE
AND iconID  is not NULL;
