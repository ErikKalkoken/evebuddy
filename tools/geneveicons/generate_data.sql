SELECT *
FROM (
	SELECT ei.iconID AS id, REPLACE(ei.iconFile, "res:/ui/texture/icons/", "") AS file
	FROM eve_sde.eveIcons ei
	WHERE POSITION("icons" IN ei.iconFile) <> 0
) as icons
WHERE SUBSTRING_INDEX(icons.file, "_", 1) REGEXP '^[0-9]+$';