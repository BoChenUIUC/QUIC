function ret = get_xy(filename)
fid = fopen(filename);
line = fgetl(fid);
x = [];
y = [];
while ischar(line)
    sts = strsplit(line);
    x = [x;str2double(sts(1))];
    y = [y;str2double(sts(2))];   
    line = fgetl(fid);
end
fclose(fid);
x = x - x(1);
ret = [x,y];
end