import com.jwoglom.pumpx2.pump.messages.Message;
import com.jwoglom.pumpx2.pump.messages.Messages;
import com.jwoglom.pumpx2.pump.messages.UndefinedMessage;
import com.jwoglom.pumpx2.pump.messages.annotations.MessageProps;
import com.jwoglom.pumpx2.pump.messages.bluetooth.Characteristic;

import org.apache.commons.lang3.tuple.Pair;
import org.json.JSONArray;
import org.json.JSONObject;

import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Modifier;
import java.lang.reflect.Parameter;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Dumps the pumpX2 message catalog the cliparser jar resolves messages against, as JSON on
 * stdout: the Messages enum in declaration order, the OPCODES table, and for every message class
 * its MessageProps, the fields its toString() prints (in ReflectionToStringBuilder's order) and
 * its public constructors in getConstructors() order, which is the order cliparser's encode
 * command tries them in.
 *
 * Run against the jar it describes:
 *   javac -cp cliparser.jar -d out CatalogDump.java && java -cp cliparser.jar:out CatalogDump
 */
public class CatalogDump {
    private static final String PREFIX = "com.jwoglom.pumpx2.pump.messages.";
    private static final Set<String> IGNORED = Set.of("requestClass", "requestProps", "historyLogStreamBytes", "intMap");

    public static void main(String[] args) throws Exception {
        JSONObject out = new JSONObject();

        JSONArray chars = new JSONArray();
        for (Characteristic c : Characteristic.values()) {
            chars.put(new JSONObject().put("name", c.name()).put("uuid", c.getUuid().toString()));
        }
        out.put("characteristics", chars);

        Set<Class<? extends Message>> classes = new LinkedHashSet<>();
        JSONArray enumEntries = new JSONArray();
        for (Messages m : Messages.values()) {
            enumEntries.put(new JSONObject()
                    .put("name", m.name())
                    .put("request", m.requestClass().getName().replace(PREFIX, ""))
                    .put("response", m.responseClass().getName().replace(PREFIX, "")));
            classes.add(m.requestClass());
            classes.add(m.responseClass());
        }
        out.put("messages", enumEntries);

        JSONArray opcodes = new JSONArray();
        List<Map.Entry<Pair<Characteristic, Integer>, Class<? extends Message>>> entries = new ArrayList<>(Messages.OPCODES.entrySet());
        entries.sort(Comparator.comparing((Map.Entry<Pair<Characteristic, Integer>, Class<? extends Message>> e) -> e.getKey().getLeft().ordinal())
                .thenComparing(e -> e.getKey().getRight()));
        for (Map.Entry<Pair<Characteristic, Integer>, Class<? extends Message>> e : entries) {
            opcodes.put(new JSONObject()
                    .put("characteristic", e.getKey().getLeft().name())
                    .put("opcode", e.getKey().getRight())
                    .put("class", e.getValue().getName().replace(PREFIX, "")));
            classes.add(e.getValue());
        }
        out.put("opcodes", opcodes);

        JSONArray classArr = new JSONArray();
        for (Class<? extends Message> c : classes) {
            classArr.put(describe(c));
        }
        out.put("classes", classArr);
        System.out.println(out.toString(1));
    }

    private static JSONObject describe(Class<? extends Message> c) throws Exception {
        JSONObject o = new JSONObject();
        o.put("name", c.getName().replace(PREFIX, ""));
        o.put("simpleName", c.getSimpleName());
        MessageProps p = c.getAnnotation(MessageProps.class);
        JSONObject props = new JSONObject();
        props.put("opCode", (int) p.opCode());
        props.put("size", p.size());
        props.put("variableSize", p.variableSize());
        props.put("stream", p.stream());
        props.put("signed", p.signed());
        props.put("type", p.type().name());
        props.put("characteristic", p.characteristic().name());
        props.put("modifiesInsulinDelivery", p.modifiesInsulinDelivery());
        props.put("minApi", p.minApi().name());
        props.put("supportedDevices", p.supportedDevices().name());
        if (!p.response().equals(UndefinedMessage.class)) {
            props.put("response", p.response().getName().replace(PREFIX, ""));
        }
        if (!p.request().equals(UndefinedMessage.class)) {
            props.put("request", p.request().getName().replace(PREFIX, ""));
        }
        o.put("props", props);

        List<String> hierarchy = new ArrayList<>();
        JSONArray fields = new JSONArray();
        Class<?> clazz = c;
        while (clazz != null) {
            hierarchy.add(clazz.getName());
            Field[] declared = clazz.getDeclaredFields();
            Arrays.sort(declared, Comparator.comparing(Field::getName));
            for (Field f : declared) {
                if (f.getName().indexOf('$') != -1) continue;
                if (Modifier.isTransient(f.getModifiers())) continue;
                if (Modifier.isStatic(f.getModifiers())) continue;
                if (IGNORED.contains(f.getName())) continue;
                fields.put(new JSONObject()
                        .put("name", f.getName())
                        .put("type", f.getGenericType().getTypeName())
                        .put("declaredIn", clazz.getName().replace(PREFIX, "")));
            }
            clazz = clazz.getSuperclass();
        }
        o.put("hierarchy", new JSONArray(hierarchy));
        o.put("fields", fields);

        JSONArray ctors = new JSONArray();
        for (Constructor<?> ctor : c.getConstructors()) {
            JSONArray params = new JSONArray();
            for (Parameter param : ctor.getParameters()) {
                params.put(new JSONObject()
                        .put("name", param.getName())
                        .put("type", param.getParameterizedType().getTypeName()));
            }
            ctors.put(new JSONObject().put("params", params));
        }
        o.put("constructors", ctors);

        // What an empty instance prints, so the field order above can be checked against the
        // real ReflectionToStringBuilder.
        try {
            Message m = c.getDeclaredConstructor().newInstance();
            m.fillWithEmptyCargo();
            o.put("emptyToString", m.toString());
        } catch (Throwable t) {
            o.put("emptyToStringError", t.toString());
        }
        return o;
    }
}
