import {sparklinePaths} from '../trafficHistory'

export default function TrafficSparkline({values, direction}: {values: number[]; direction: 'rx' | 'tx'}) {
  const {line, area} = sparklinePaths(values)
  if (!line) return null
  return (
    <svg
      className={'traffic-sparkline ' + direction}
      viewBox="0 0 100 40"
      preserveAspectRatio="none"
      aria-hidden="true"
      focusable="false"
    >
      <path className="traffic-area" d={area} fill="currentColor" opacity="0.3" />
      <path
        className="traffic-line"
        d={line}
        pathLength={1}
        fill="none"
        stroke="currentColor"
        strokeWidth="1.3"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  )
}
