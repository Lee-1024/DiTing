import { Select } from 'antd';
import type { SelectProps } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { getHostAudits, getUserAudits } from '../api/stats';
import { buildHostOptions, buildUserOptions } from './auditEntityOptions';
import type { AuditSelectOption } from './auditEntityOptions';

interface AuditEntitySelectProps extends Omit<SelectProps<string>, 'loading' | 'options' | 'showSearch'> {
  startTime?: string;
  endTime?: string;
}

type AuditEntityKind = 'user' | 'host';

function AuditEntitySelect({
  kind,
  startTime,
  endTime,
  ...selectProps
}: AuditEntitySelectProps & { kind: AuditEntityKind }) {
  const [options, setOptions] = useState<AuditSelectOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const requestSeq = useRef(0);

  useEffect(() => {
    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    setLoading(true);
    setFailed(false);

    const request = kind === 'user'
      ? getUserAudits({ start_time: startTime, end_time: endTime, limit: 100 }).then(buildUserOptions)
      : getHostAudits({ start_time: startTime, end_time: endTime, limit: 100 }).then(buildHostOptions);

    void request.then((nextOptions) => {
      if (seq === requestSeq.current) {
        setOptions(nextOptions);
      }
    }).catch(() => {
      if (seq === requestSeq.current) {
        setOptions([]);
        setFailed(true);
      }
    }).finally(() => {
      if (seq === requestSeq.current) {
        setLoading(false);
      }
    });

    return () => {
      requestSeq.current += 1;
    };
  }, [endTime, kind, startTime]);

  return (
    <Select<string>
      {...selectProps}
      allowClear
      showSearch
      optionFilterProp="label"
      loading={loading}
      options={options}
      notFoundContent={loading ? '正在加载...' : failed ? '选项加载失败' : '暂无审计数据'}
    />
  );
}

export function AuditUserSelect(props: AuditEntitySelectProps) {
  return <AuditEntitySelect {...props} kind="user" placeholder={props.placeholder ?? '选择用户'} />;
}

export function AuditHostSelect(props: AuditEntitySelectProps) {
  return <AuditEntitySelect {...props} kind="host" placeholder={props.placeholder ?? '选择主机'} />;
}
