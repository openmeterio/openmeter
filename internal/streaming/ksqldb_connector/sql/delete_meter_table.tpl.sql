DROP TABLE {{ printf "OM_%s_METER_%s" .Namespace .Slug | upper | bquote  }} DELETE TOPIC;
