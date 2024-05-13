import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface SchedulerIconProps extends SVGIconProps {
    scale?: number;
}

export class SchedulerIcon extends React.Component<SchedulerIconProps> {
    static displayName = 'SchedulerIcon';

    id = `icon-title-scheduler-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v5-svg ${className}` : 'pf-v5-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 0, 100, 100].join(' ');

        return (
            <svg
                className={classes}
                viewBox={viewBox}
                fill="currentColor"
                aria-labelledby={hasTitle ? this.id : undefined}
                aria-hidden={hasTitle ? undefined : true}
                role="img"
                width="1em"
                height="1em"
                transform={`scale(${scale})`}
                {...(props as Omit<React.SVGProps<SVGElement>, 'ref'>)} // Lie.
            >
                {hasTitle && <title id={this.id}>{title}</title>}
                <path d="M79.67,49.23l-10-10a1.36,1.36,0,0,0-1.86,0l-1.86,1.87a1.34,1.34,0,0,0,0,1.86L69,46a.89.89,0,0,1-.09,1.27.87.87,0,0,1-.53.22H41L57.33,31.24a.89.89,0,0,1,1.27.09.87.87,0,0,1,.22.53v4.46a1.18,1.18,0,0,0,1.1,1.24h2.71a1.17,1.17,0,0,0,1.24-1.09V22a1.17,1.17,0,0,0-1.1-1.24H48.53a1.36,1.36,0,0,0-1.24,1.37v2.49a1.18,1.18,0,0,0,1.12,1.24H53a.88.88,0,0,1,.86.9.89.89,0,0,1-.24.58L33.52,47.37H21.24A1.28,1.28,0,0,0,20,48.69v2.64a1.46,1.46,0,0,0,1.36,1.37H33.77l20,20a.9.9,0,0,1-.1,1.27.85.85,0,0,1-.52.21h-4.5a1.17,1.17,0,0,0-1.24,1.1.33.33,0,0,0,0,.14v2.49a1.45,1.45,0,0,0,1.24,1.37H62.8A1.18,1.18,0,0,0,64,78.16V63.74a1.18,1.18,0,0,0-1.12-1.24H60.18a1.17,1.17,0,0,0-1.24,1.09V68.2a.88.88,0,0,1-.89.87.9.9,0,0,1-.6-.25L41.08,52.7H68.5a.89.89,0,0,1,.62,1.48l-3.22,3.1a1.35,1.35,0,0,0,0,1.87L67.76,61a1.34,1.34,0,0,0,1.86,0l10-9.92A1.36,1.36,0,0,0,79.67,49.23Z" />
            </svg>
        );
    }
}
