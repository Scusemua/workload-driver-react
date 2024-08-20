import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface TemplateIconProps extends SVGIconProps {
    scale?: number;
}

export class TemplateIcon extends React.Component<TemplateIconProps> {
    static displayName = 'TemplateIcon';

    id = `icon-title-template-icon`;

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
                <path
                    d="M65.7,34h7.9c0.9,0,1.7-0.8,1.7-1.7c0-0.4-0.2-0.9-0.5-1.2L63.5,20c-0.3-0.3-0.7-0.5-1.1-0.5
                    c-0.9,0-1.7,0.8-1.7,1.7v7.9C60.9,31.8,63,33.9,65.7,34z"
                />
                <path
                    d="M72.7,41H60.5c-4,0-7.2-3.2-7.3-7.2v-12c0.1-1.2-0.8-2.2-2-2.3c-0.1,0-0.2,0-0.3,0H31.6
                    c-4,0-7.2,3.2-7.3,7.2v45.7c0,4,3.3,7.2,7.3,7.2h36.2c4,0,7.2-3.2,7.3-7.2V43.5C75.2,42.2,74.1,41,72.7,41C72.7,41,72.7,41,72.7,41z
                    M65.8,71.8C65,72.6,64,73,62.8,73c-1.1,0-2.2-0.4-3-1.2L46,58c-0.9,0.4-1.8,0.6-2.8,0.7c-5.8,0.7-11-3.5-11.7-9.3
                    c-0.2-1.5,0-3,0.4-4.4c0.1-0.4,0.6-0.5,1-0.2l6,6c0.4,0.5,1.2,0.5,1.6,0.1c0,0,0.1,0,0.1-0.1l4.2-4.2c0.5-0.4,0.5-1.2,0.1-1.6
                    c0,0,0-0.1-0.1-0.1l-6-6c-0.2-0.2-0.2-0.6,0-0.9c0.1-0.1,0.1-0.1,0.2-0.1c1-0.2,2.1-0.4,3.1-0.4c5.9,0,10.6,4.7,10.7,10.6
                    c0,0.4,0,0.8-0.1,1.2c-0.1,1-0.4,1.9-0.7,2.8L65.8,66C67.4,67.6,67.4,70.2,65.8,71.8L65.8,71.8z"
                />
            </svg>
        );
    }
}