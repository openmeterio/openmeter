export const getQueryGroupedProjectionQuery = (
  tableName: string,
  projectionValueSQL: string,
  valueSQL: string,
  whereSQL: string
): string => {
  return `
  WITH proj AS (
    SELECT
        subject,
        tumbleStart(
          time,
          toIntervalMinute(1)
        ) AS windowstart,
        tumbleEnd(
          time,
          toIntervalMinute(1)
        ) AS windowend,
        ${projectionValueSQL} AS value,
        JSON_VALUE(data, '$.group1') as group1,
        JSON_VALUE(data, '$.group2') as group2
    FROM
      ${tableName}
    WHERE
      (${tableName}.namespace = 'default')
      AND (empty(${tableName}.validation_error) = 1)
      AND (${tableName}.type = 'etype')
    GROUP BY
      windowstart,
      windowend,
      subject,
      group1,
      group2
  ) SELECT windowstart, windowend, ${valueSQL} AS value, subject, group1, group2
  FROM proj
  ${whereSQL}
  GROUP BY
    windowstart,
    windowend,
    subject,
    group1,
    group2
  ORDER BY
    windowstart`;
};
