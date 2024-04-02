// import { useEffect, useRef } from 'react';

// function useInterval(callback, delay) {
//     const savedCallback = useRef();

//     // Remember the latest callback.
//     useEffect(() => {
//         savedCallback.current = callback;
//     }, [callback]);

//     // Set up the interval.
//     useEffect(() => {
//         function tick() {
//             if (savedCallback.current) {
//                 savedCallback.current();
//             }
//         }
//         if (delay !== null) {
//             let id = setInterval(tick, delay);
//             return () => clearInterval(id);
//         }
//         return;
//     }, [delay]);
// }
