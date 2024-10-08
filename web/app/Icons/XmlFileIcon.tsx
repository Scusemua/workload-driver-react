import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface XmlFileIconProps extends SVGIconProps {
    scale?: number;
}

export class XmlFileIcon extends React.Component<XmlFileIconProps> {
    static displayName = 'XmlFileIcon';

    id = `icon-title-xml-file-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v5-svg ${className}` : 'pf-v5-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 0, 58, 58].join(' ');

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
                <g>
                    <path
                        d="M51.5,39V13.978c0-0.766-0.092-1.333-0.55-1.792L39.313,0.55C38.964,0.201,38.48,0,37.985,0H8.963
		C7.777,0,6.5,0.916,6.5,2.926V39H51.5z M37.5,3.391c0-0.458,0.553-0.687,0.877-0.363l10.095,10.095
		C48.796,13.447,48.567,14,48.109,14H37.5V3.391z M33.793,18.707c-0.391-0.391-0.391-1.023,0-1.414s1.023-0.391,1.414,0l6,6
		c0.391,0.391,0.391,1.023,0,1.414l-6,6C35.012,30.902,34.756,31,34.5,31s-0.512-0.098-0.707-0.293
		c-0.391-0.391-0.391-1.023,0-1.414L39.086,24L33.793,18.707z M24.557,31.667l6-17c0.185-0.521,0.753-0.795,1.276-0.61
		c0.521,0.184,0.794,0.755,0.61,1.276l-6,17C26.298,32.744,25.912,33,25.5,33c-0.11,0-0.223-0.019-0.333-0.058
		C24.646,32.759,24.373,32.188,24.557,31.667z M15.793,23.293l6-6c0.391-0.391,1.023-0.391,1.414,0s0.391,1.023,0,1.414L17.914,24
		l5.293,5.293c0.391,0.391,0.391,1.023,0,1.414C23.012,30.902,22.756,31,22.5,31s-0.512-0.098-0.707-0.293l-6-6
		C15.402,24.316,15.402,23.684,15.793,23.293z"
                    />
                    <path
                        d="M6.5,41v15c0,1.009,1.22,2,2.463,2h40.074c1.243,0,2.463-0.991,2.463-2V41H6.5z M22.936,54h-1.9l-1.6-3.801h-0.137
		L17.576,54h-1.9l2.557-4.895l-2.721-5.182h1.873l1.777,4.102h0.137l1.928-4.102H23.1l-2.721,5.182L22.936,54z M34.666,54h-1.668
		v-6.932l-2.256,5.605h-1.449l-2.27-5.605V54h-1.668V43.924h1.668l2.994,6.891l2.98-6.891h1.668V54z M43.498,54h-6.303V43.924h1.668
		v8.832h4.635V54z"
                    />
                </g>
            </svg>
        );
    }
}
