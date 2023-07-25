DESCRIBE {{ printf "OM_METER_%s" .Slug | upper | bquote  }} EXTENDED;
