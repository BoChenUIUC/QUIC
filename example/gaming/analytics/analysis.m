frame = get_xy('test3/frame.dat');

probe = get_xy('test3/probe.dat');

xack = get_xy('test3/xack.dat');

hold on
plot(frame(:,1),frame(:,2),'r','LineWidth',2)
plot(probe(:,1),probe(:,2),'g','LineWidth',2)
plot(xack(:,1),xack(:,2),'b','LineWidth',2)
legend('transmission latency','probing latency','xack latency')