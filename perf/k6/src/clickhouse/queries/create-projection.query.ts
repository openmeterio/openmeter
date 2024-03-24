export const getCreateGroupedProjectionQuery = (
  tableName: string,
  projectionName: string,
  valueSQL: string
): string => `
ALTER TABLE ${tableName}
ADD PROJECTION IF NOT EXISTS ${projectionName} (
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
      ${valueSQL} AS value,
      JSON_VALUE(data, '$.group1') as group1,
      JSON_VALUE(data, '$.group2') as group2,
      namespace,
      empty(validation_error) as isvalid,
      type
    GROUP BY
      windowstart,
      windowend,
      subject,
      group1,
      group2,
      namespace,
      isvalid,
      type
  );
`;

export const getMaterialiseProjectionQuery = (tableName: string, projectionName: string): string => `
ALTER TABLE ${tableName} MATERIALIZE PROJECTION ${projectionName};
`;
