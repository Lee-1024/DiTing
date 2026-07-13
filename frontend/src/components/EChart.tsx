import * as echarts from 'echarts';
import { useEffect, useRef } from 'react';

interface Props {
  option: echarts.EChartsOption;
  height?: number;
}

export default function EChart({ option, height = 280 }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) {
      return;
    }
    const chart = echarts.init(ref.current);
    chart.setOption(option);
    const resize = () => chart.resize();
    window.addEventListener('resize', resize);
    return () => {
      window.removeEventListener('resize', resize);
      chart.dispose();
    };
  }, [option]);

  return <div style={{ width: '100%', height }} ref={ref} />;
}
