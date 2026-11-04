export type IconName =
  'power' | 'stop' | 'servers' | 'logs' | 'settings' | 'plus' | 'arrow' | 'download' | 'upload' | 'tray' | 'shield'

const paths: Record<IconName, string> = {
  power: 'M12 3v9 M6.3 5.8a8 8 0 1 0 11.4 0',
  stop: 'M8 6h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2z',
  servers: 'M4 4h16v6H4z M4 14h16v6H4z M7 7h.01 M7 17h.01 M11 7h6 M11 17h6',
  logs: 'M5 4h14v16H5z M8 8h8 M8 12h8 M8 16h5',
  settings: 'M9 3h6l1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1z M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0',
  plus: 'M12 5v14 M5 12h14',
  arrow: 'M5 12h14 M13 6l6 6-6 6',
  download: 'M12 3v12 M7 10l5 5 5-5 M4 17v4h16v-4',
  upload: 'M12 15V3 M7 8l5-5 5 5 M4 17v4h16v-4',
  tray: 'M4 4h16v11H4z M8 20h8 M12 15v5 M9 8l3 3 3-3',
  shield: 'M12 3l8 3v6c0 5-8 9-8 9s-8-4-8-9V6z M8 12l3 3 5-6',
}

export default function Icon({name, size = 20}: {name: IconName; size?: number}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path
        d={paths[name]}
        fill={name === 'stop' ? 'currentColor' : 'none'}
        stroke={name === 'stop' ? 'none' : 'currentColor'}
      />
    </svg>
  )
}
