import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import React from 'react';

interface Ds2IconProps extends SVGIconProps {
    scale?: number;
}

export class Ds2Icon extends React.Component<Ds2IconProps> {
    static displayName = 'Ds2Icon';

    id = `icon-title-ds2-icon`;

    render() {
        let scale = 1;
        if (this.props.scale) {
            scale = this.props.scale;
        }

        const { title, className, ...props } = this.props;
        const classes = className ? `pf-v6-svg ${className}` : 'pf-v6-svg';

        const hasTitle = Boolean(title);
        const viewBox = [0, 0, 305, 209].join(' ');

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
                <g transform="translate(0.000000,290.000000) scale(0.100000,-0.100000)" stroke="none" fill="#ffff">
                    <path
                        d="M190 1470 l0 -1210 1210 0 1210 0 0 1210 0 1210 -1210 0 -1210 0 0
-1210z m2147 995 c90 -44 125 -117 125 -260 1 -100 -15 -171 -68 -308 -33 -86
-133 -292 -201 -414 l-35 -63 146 0 146 0 0 -75 0 -75 -265 0 c-146 0 -265 4
-265 8 0 5 58 130 129 279 207 432 241 523 241 649 0 88 -23 130 -75 140 -76
14 -113 -46 -114 -184 l-1 -83 -82 3 -83 3 1 95 c2 153 40 247 115 285 80 42
200 41 286 0z m-649 -417 l62 -20 0 -104 c0 -57 -1 -104 -2 -104 -2 0 -30 14
-62 30 -104 53 -194 35 -247 -49 -25 -38 -29 -55 -29 -114 0 -96 22 -136 157
-278 189 -200 237 -279 252 -411 26 -237 -75 -417 -260 -464 -90 -23 -173 -15
-255 26 l-64 31 0 98 c0 54 3 100 6 103 3 3 34 -9 69 -27 86 -45 140 -53 197
-27 105 46 136 186 72 320 -16 33 -67 96 -137 170 -147 154 -181 196 -214 268
-26 55 -28 68 -28 189 0 125 1 133 31 195 81 171 248 233 452 168z m-948 -14
c156 -46 253 -192 291 -439 17 -108 17 -506 0 -610 -57 -345 -185 -444 -578
-445 l-143 0 0 755 0 755 188 0 c144 0 201 -4 242 -16z"
                    />
                    <path
                        d="M527 1874 c-4 -4 -7 -264 -7 -577 l0 -570 58 6 c112 10 178 64 212
176 43 136 50 495 14 699 -27 153 -90 242 -183 257 -20 4 -47 8 -61 11 -14 3
-29 2 -33 -2z"
                    />
                </g>
            </svg>
        );
    }
}
