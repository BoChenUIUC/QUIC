rootdir = "QUIC/";
frame = get_xy(rootdir + "frame.dat");
mean(frame(:,2))
std(frame(:,2))

probe = get_xy(rootdir + "probe.dat");

xack  = get_xy(rootdir + "xack.dat");
xack(xack==100)=-0.01;

dlmwrite(rootdir+"frame_latency.txt",frame)
dlmwrite(rootdir+"server_avg_latency.txt",xack)

hold on
plot(frame(:,1),frame(:,2),'r','LineWidth',2)
plot(probe(:,1),probe(:,2),'g','LineWidth',2)
plot(xack(:,1),xack(:,2),'b','LineWidth',2)
legend('transmission latency','probing latency','xack latency')